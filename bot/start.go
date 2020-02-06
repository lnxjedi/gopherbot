// +build linux darwin dragonfly freebsd netbsd openbsd

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
	flag.Parse()
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
		Log(robot.Info, "quitting on child exit")
		return
	}

	usage := `Usage: gopherbot [options] [command [command options]]
  "command" can be one of:
	encrypt - encrypt a string or file
	decrypt - decrypt a string or file
	list - list robot memories
	delete - delete a memory
	fetch - fetch the contents of a memory
	store - store a memory
	run - run the robot (default)
	dump (installed|configured) [path/to/file.yaml] -
	  read and dump a raw config file, for yaml troubleshooting
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

	// terminal mode overrides any other setting of GOPHER_PROTOCOL
	if terminalmode {
		os.Setenv("GOPHER_PROTOCOL", "terminal")
	}

	envCfgPath := os.Getenv("GOPHER_CONFIGDIR")
	// Configdir is where all user-supplied configuration and
	// external plugins are.
	if len(explicitCfgPath) != 0 {
		configpath = explicitCfgPath
	} else if len(envCfgPath) > 0 {
		configpath = envCfgPath
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
	testpath := filepath.Join(configpath, "conf", robotConfigFileName)
	_, err := os.Stat(testpath)
	if err != nil {
		testpath = filepath.Join(configpath, "conf", "gopherbot.yaml")
		_, err = os.Stat(testpath)
		if err == nil {
			robotConfigFileName = "gopherbot.yaml"
		}
	}
	if err != nil {
		_, ok := os.LookupEnv("GOPHER_CUSTOM_REPOSITORY")
		if !ok {
			Log(robot.Warn, "Starting unconfigured; no robot.yaml/gopherbot.yaml found")
			os.Setenv("GOPHER_UNCONFIGURED", "unconfigured")
			os.Setenv("GOPHER_PROTOCOL", "terminal")
			if _, ok := os.LookupEnv("GOPHER_LOGFILE"); !ok {
				os.Setenv("GOPHER_LOGFILE", "robot.log")
				defaultLogfile = true
			}
		} else {
			// no robot.yaml, but GOPHER_CUSTOM_REPOSITORY set
			os.Setenv("GOPHER_PROTOCOL", "nullconn")
		}
		defaultProto = true
	} else {
		os.Unsetenv("GOPHER_UNCONFIGURED")
	}

	var logger *log.Logger
	logOut := os.Stdout
	botStdOutLogging = true
	if len(logFile) == 0 {
		logFile = os.Getenv("GOPHER_LOGFILE")
	}
	eproto := os.Getenv("GOPHER_PROTOCOL")
	if len(logFile) == 0 && (cliOp || eproto == "terminal") {
		logFile = "robot.log"
	}
	if len(logFile) != 0 {
		if logFile == "stderr" {
			botStdOutLogging = false
			logOut = os.Stderr
		} else {
			lf, err := os.Create(logFile)
			if err != nil {
				log.Fatalf("Error creating log file: (%T %v)", err, err)
			}
			botStdOutLogging = false
			fileLog = true
			logFileName = logFile
			logOut = lf
		}
	}
	// Not needed?
	// log.SetOutput(logOut)
	logger = log.New(logOut, "", logFlags)
	botLogger.l = logger
	if fileLog {
		botLogger.setOutputFile(logOut)
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
		Log(robot.Info, "backgrounding")
		bin, _ := os.Executable()
		env := os.Environ()
		if !fileLog {
			env = append(env, "GOPHER_LOGFILE=robot.log")
		} else {
			env = append(env, fmt.Sprintf("GOPHER_LOGFILE=%s", logFile))
		}
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

	initializeConnector, ok := connectors[currentCfg.protocol]
	if !ok {
		logger.Fatalf("No connector registered with name: %s", currentCfg.protocol)
	}
	if currentCfg.protocol == "terminal" {
		local = true
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
			os.Unsetenv("GOPHER_PROTOCOL")
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
