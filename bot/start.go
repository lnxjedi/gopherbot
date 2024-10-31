package bot

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

// Information about privilege separation, set in runtasks_linux.go
var privSep = false

// Set for CLI commands
var cliOp = false
var fileLog = false

func init() {
	hostName = os.Getenv("HOSTNAME")
}

// Start gets the robot going
// SEE ALSO: start_t.go
func Start(v VersionInfo) {
	botVersion = v

	var configpath string

	// Save args in case we need to spawn child
	args := os.Args[1:]
	// Process command-line flags
	var explicitCfgPath string
	cusage := "path to the configuration directory"
	flag.StringVar(&explicitCfgPath, "config", "", cusage)
	flag.StringVar(&explicitCfgPath, "c", "", "")
	var daemonize bool
	var dusage = "daemonize on startup"
	flag.BoolVar(&daemonize, "daemonize", false, dusage)
	flag.BoolVar(&daemonize, "d", false, "")
	var logFile string
	lusage := "path to robot's log file (or 'stderr')"
	flag.StringVar(&logFile, "log", "", lusage)
	flag.StringVar(&logFile, "l", "", "")
	var overrideIDEMode bool
	ovusage := "Override GOPHER_IDE mode"
	flag.BoolVar(&overrideIDEMode, "override", false, ovusage)
	flag.BoolVar(&overrideIDEMode, "o", false, "")
	var plainlog bool
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "p", false, "")
	var terminalmode bool
	tmusage := "set 'GOPHER_PROTOCOL=terminal' and default logging to 'robot.log'"
	flag.BoolVar(&terminalmode, "terminal", false, tmusage)
	flag.BoolVar(&terminalmode, "t", false, "")
	var help bool
	husage := "help for gopherbot"
	flag.BoolVar(&help, "help", false, husage)
	flag.BoolVar(&help, "h", false, "")
	// TODO: Gopherbot CLI commands suck. Make them suck less.
	flag.Parse()

	_, ideMode := os.LookupEnv("GOPHER_IDE")
	if ideMode {
		// To prevent inadvertently bootstrapping a production
		// robot in a random directory, we force it to run in $HOME.
		// This behavior can only be overridden by unsetting
		// GOPHER_IDE in the terminal before invoking gopherbot.
		// (`unset GOPHER_IDE; gopherbot`)
		homeDir := os.Getenv("HOME")
		os.Chdir(homeDir)
		if overrideIDEMode {
			ideMode = false
		} else {
			// Guardrail when running the gopherbot IDE from cbot.sh, which sets
			// GOPHER_IDE=true.  with
			// the terminal connector and memory brain, unless the user specifically
			// overrides this behavior.
			_, profileConfigured := os.LookupEnv("GOPHER_CUSTOM_REPOSITORY")
			termEnv := os.Getenv("GOPHER_PROTOCOL") == "terminal"
			if profileConfigured && !termEnv {
				os.Setenv("GOPHER_PROTOCOL", "terminal")
				os.Setenv("GOPHER_BRAIN", "mem")
			} else {
				ideMode = false
			}
		}
	}

	logFlags := log.LstdFlags
	if plainlog {
		logFlags = 0
	}
	botStdErrLogger = log.New(os.Stderr, "", logFlags)
	botStdOutLogger = log.New(os.Stdout, "", logFlags)
	// Container support
	pid := os.Getpid()
	if pid == 1 {
		Log(robot.Info, "PID == 1, spawning child")
		bin, _ := os.Executable()
		os.Setenv("GOPHER_CONTAINER", "iscontainer")
		env := os.Environ()
		cmd := exec.Command(bin, args...)
		cmd.Env = env
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		raiseThreadPrivExternal("exec child process")
		err := cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		go initSigHandle(cmd.Process)
		if pid == 1 {
			go func() {
				var ws unix.WaitStatus
				// Reap children FOREVER...
				for {
					pid, err := unix.Wait4(-1, &ws, 0, nil)
					if err == nil {
						Log(robot.Debug, "Reaped child pid: %d, status: %+v", pid, ws)
					}
				}
			}()
		}
		cmd.Wait()
		Log(robot.Info, "Quitting on child exit")
		return
	}

	usage := `Usage: gopherbot [options] [command [command options] [command args]]
  "command" can be one of:
	decrypt - decrypt a string or file
	encrypt - encrypt a string or file
	gentotp - generate a user TOTP string
	delete - delete a memory
	dump (installed|configured) [path/to/file.yaml] -
	  read and dump a raw config file, for yaml troubleshooting
	fetch - fetch the contents of a memory
	init (protocol) - create a new robot in currect directory
	list - list robot memories
	run - run the robot (default)
	store - store a memory
	version - display the gopherbot version
  <command> -h for help on a given command

  Common options:`

	if help {
		fmt.Println(usage)
		flag.PrintDefaults()
		os.Exit(0)
	}

	cliOp = len(flag.Args()) > 0 && flag.Arg(0) != "run"
	var cliCommand string
	if cliOp {
		cliCommand = flag.Arg(0)
	}

	var envFile string
	var fixed = []string{}
	for _, ef := range []string{"private/environment", ".env"} {
		if es, err := os.Stat(ef); err == nil {
			em := es.Mode()
			if (uint32(em) & 0077) != 0 {
				mask := os.FileMode(0700)
				want := em & mask
				if err := os.Chmod(ef, want); err != nil {
					log.Fatalf("Invalid file mode '%o' on environment file '%s', can't fix: %v", em, ef, err)
				}
				fixed = append(fixed, ef)
			}
			envFile = ef
		}
	}
	penvErr := godotenv.Overload(envFile)

	if _, ok := os.LookupEnv("GOPHER_ENVIRONMENT"); !ok {
		os.Setenv("GOPHER_ENVIRONMENT", "production")
	}

	// Configdir is where all user-supplied configuration and
	// external plugins are.
	if len(explicitCfgPath) != 0 {
		configpath = explicitCfgPath
	} else {
		if _, ok := checkDirectory("custom"); ok {
			configpath = "custom"
		} else if _, ok := checkDirectory("conf"); ok {
			configpath = "."
		} else {
			// If not explicitly set or cwd, use "custom" even if it
			// doesn't exist.
			configpath = "custom"
		}
	}

	// support for setup and bootstrap plugins
	var defaultProto, defaultLogfile bool

	termStart := func() {
		defaultProto = true
		os.Setenv("GOPHER_PROTOCOL", "terminal")
		if _, ok := os.LookupEnv("GOPHER_LOGFILE"); !ok {
			os.Setenv("GOPHER_LOGFILE", "robot.log")
			defaultLogfile = true
		}
	}

	protoEnv, protoSet := os.LookupEnv("GOPHER_PROTOCOL")
	testpath := filepath.Join(configpath, "conf", robotConfigFileName)
	_, err := os.Stat(testpath)
	if err != nil {
		testpath = filepath.Join(configpath, "conf", "gopherbot.yaml")
		_, err = os.Stat(testpath)
		if err == nil {
			robotConfigFileName = "gopherbot.yaml"
		}
	}
	unconfigured := false
	if err != nil {
		_, ok := os.LookupEnv("GOPHER_CUSTOM_REPOSITORY")
		if !ok {
			unconfigured = true
			os.Setenv("GOPHER_UNCONFIGURED", "unconfigured")
			// Start a setup plugin; if answerfile.txt is present, or ANS_PROTOCOL is set,
			// use the new-style, otherwise run the terminal connector for the interactive plugin.
			setup := false
			if _, err := os.Stat("answerfile.txt"); err == nil {
				// true for CLI setup
				setup = true
			} else if _, ok := os.LookupEnv("ANS_PROTOCOL"); ok {
				// true for container-based setup
				setup = true
			}
			if setup {
				defaultProto = true
				os.Setenv("GOPHER_PROTOCOL", "nullconn")
				if _, ok := os.LookupEnv("GOPHER_LOGFILE"); !ok {
					os.Setenv("GOPHER_LOGFILE", "robot.log")
					if !cliOp {
						Log(robot.Info, "Logging to robot.log")
					}
					defaultLogfile = true
				}
			} else {
				termStart()
			}
		} else {
			// no robot.yaml, but GOPHER_CUSTOM_REPOSITORY set
			os.Setenv("GOPHER_PROTOCOL", "nullconn")
		}
		defaultProto = true
	} else {
		os.Unsetenv("GOPHER_UNCONFIGURED")
		if !protoSet || terminalmode {
			termStart()
		}
	}

	// Set up Logging
	var logger *log.Logger
	logOut := os.Stdout
	if len(logFile) == 0 {
		logFile = os.Getenv("GOPHER_LOGFILE")
	}
	eproto := os.Getenv("GOPHER_PROTOCOL")
	if len(logFile) == 0 && (cliOp || eproto == "terminal") {
		logFile = "robot.log"
	}
	if len(logFile) != 0 {
		if logFile == "stderr" {
			logOut = os.Stderr
		} else {
			lf, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("Error creating log file: (%T %v)", err, err)
			}
			fileLog = true
			logFileName = logFile
			logOut = lf
		}
	}

	logger = log.New(logOut, "", logFlags)
	botLogger.logger = logger
	if unconfigured {
		Log(robot.Warn, "Starting unconfigured; no robot.yaml/gopherbot.yaml found")
	}
	if ideMode {
		Log(robot.Info, "Starting in IDE mode, defaulting GOPHER_BRAIN to 'mem' and GOPHER_PROTOCOL to 'terminal'; override with '-o' flag")
	}

	if daemonize {
		scrubargs := []string{}
		skip := false
		for _, arg := range args {
			if arg == "-d" || arg == "-daemonize" {
				continue
			}
			if arg == "-l" || arg == "-log" {
				skip = true
				continue
			}
			if skip {
				skip = false
				continue
			}
			scrubargs = append(scrubargs, arg)
		}
		bin, _ := os.Executable()
		env := os.Environ()
		if !fileLog {
			Log(robot.Info, "Logging to robot.log")
			env = append(env, "GOPHER_LOGFILE=robot.log")
		} else {
			env = append(env, fmt.Sprintf("GOPHER_LOGFILE=%s", logFile))
		}
		Log(robot.Info, "Forking in to background...")
		cmd := exec.Command(bin, scrubargs...)
		cmd.Env = env
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		raiseThreadPrivExternal("fork in to background")
		err := cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if !cliOp {
		lle := os.Getenv("GOPHER_LOGLEVEL")
		if len(lle) > 0 {
			loglevel := logStrToLevel(lle)
			setLogLevel(loglevel)
		}
		logger.Println("Initialized logging ...")
	}

	if !cliOp {
		if penvErr != nil {
			logger.Printf("No private environment loaded from '.env': %v\n", penvErr)
		} else {
			logger.Printf("Loaded initial private environment from '%s'\n", envFile)
		}
		if len(fixed) > 0 {
			logger.Printf("Notice! Fixed invalid file modes for environment file(s): %s", strings.Join(fixed, ", "))
		}

		// Create the 'bot and load configuration, supplying configpath and installpath.
		// When loading configuration, gopherbot first loads default configuration
		// from internal config, then loads from configpath/conf/..., which
		// overrides defaults.
		logger.Printf("Starting up with config dir: %s, and install dir: %s\n", configpath, binDirectory)
		checkprivsep(logger)
	}

	if cliCommand == "dump" {
		setLogLevel(robot.Warn)
		if len(flag.Args()) != 3 {
			fmt.Println(usage)
			flag.PrintDefaults()
			os.Exit(1)
		}
		switch flag.Arg(1) {
		case "installed", "configured":
			configPath = configpath
			installPath = binDirectory
			initCrypt()
			cliDump(flag.Arg(1), flag.Arg(2))
		default:
			fmt.Println("DEBUG default")
			fmt.Println(usage)
			flag.PrintDefaults()
			os.Exit(1)
		}
	}

	initBot(configpath, binDirectory)

	if cliOp {
		go runBrain()
		processCLI(usage)
		brainQuit()
		return
	}
	if currentCfg.protocol == "terminal" {
		localTerm = true
		if defaultLogfile {
			botStdOutLogger.Println("Logging to robot.log; warnings and errors duplicated to stdout")
		}
	}
	if currentCfg.protocol == "nullconn" {
		nullConn = true
	}
	initializeConnector, ok := connectors[currentCfg.protocol]
	if !ok {
		logger.Fatalf("No connector registered with name: %s", currentCfg.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	conn := initializeConnector(handle, logger)

	// NOTE: we use setConnector instead of passing the connector to run()
	// because of the way Windows services were run. Maybe remove eventually?
	setConnector(conn)

	// Start the robot loops
	run()
	// ... and wait for the robot to stop
	restart := <-done
	raiseThreadPrivExternal("Exiting")
	time.Sleep(time.Second)
	if restart {
		if defaultProto {
			if protoSet {
				os.Setenv("GOPHER_PROTOCOL", protoEnv)
			} else {
				os.Unsetenv("GOPHER_PROTOCOL")
			}
		}
		if defaultLogfile {
			os.Unsetenv("GOPHER_LOGFILE")
		}
		bin, _ := os.Executable()
		env := os.Environ()
		defer func() {
			err := unix.Exec(bin, os.Args, env)
			if err != nil {
				fmt.Printf("Error re-exec'ing: %v", err)
			}
		}()
	}
}
