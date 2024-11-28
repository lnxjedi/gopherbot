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

var (
	// Set for CLI commands
	cliOp   = false
	fileLog = false

	// CLI flags
	logFile         string
	overrideIDEMode bool
	plainlog        bool
	helpRequested   bool

	hostName string
)

func init() {
	hostName = os.Getenv("HOSTNAME")

	// loggers of last resort, initialize early and update in start.go
	botStdErrLogger = log.New(os.Stderr, "", log.LstdFlags)
	botStdOutLogger = log.New(os.Stdout, "", log.LstdFlags)
}

// Start gets the robot going
// SEE ALSO: start_t.go for "make test"
func Start(v VersionInfo) {
	botVersion = v

	var err error
	// Installpath is where the default config and stock external
	// plugins are.
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	installPath, err = filepath.Abs(filepath.Dir(ex))
	if err != nil {
		panic(err)
	}

	// Go ahead and initialize privsep, needed for container support below
	initializePrivsep()

	// Save args in case we need to spawn child
	args := os.Args[1:]
	// Process command-line flags
	lusage := "path to robot's log file (or 'stderr')"
	flag.StringVar(&logFile, "log", "", lusage)
	flag.StringVar(&logFile, "l", "", "")
	ovusage := "Override GOPHER_IDE mode"
	flag.BoolVar(&overrideIDEMode, "override", false, ovusage)
	flag.BoolVar(&overrideIDEMode, "o", false, "")
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "p", false, "")
	husage := "help for gopherbot"
	flag.BoolVar(&helpRequested, "help", false, husage)
	flag.BoolVar(&helpRequested, "h", false, "")
	// TODO: Gopherbot CLI commands suck. Make them suck less.
	flag.Parse()

	/*
		To prevent inadvertently bootstrapping a production
		robot in a random directory, we force it to run in $HOME
		when in IDE mode. This behavior can only be overridden by
		unsetting GOPHER_IDE in the terminal before invoking gopherbot.
		(`unset GOPHER_IDE; gopherbot`).

		IDE mode also ensures that the protocol is "terminal" and the
		brain is "mem" (in-memory); these can be overridden with the "-o"
		CLI flag.
	*/
	_, ideMode := os.LookupEnv("GOPHER_IDE")
	if ideMode {
		homeDir := os.Getenv("HOME")
		os.Chdir(homeDir)
		if overrideIDEMode {
			ideMode = false
		} else {
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
		raisePrivPermanent("exec child process for container")
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
	validate [path/to/robot_repo] - syntax check a robot's repository
	version - display the gopherbot version
  <command> -h for help on a given command

  Common options:`

	if helpRequested {
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

	// Support for dev environment with alternate encryption keys.
	if _, ok := os.LookupEnv("GOPHER_ENVIRONMENT"); !ok {
		os.Setenv("GOPHER_ENVIRONMENT", "production")
	}

	// support for setup and bootstrap plugins
	var defaultProto, defaultLogfile bool

	termStart := func() {
		defaultProto = true
		os.Setenv("GOPHER_PROTOCOL", "terminal")
	}

	protoEnv, protoSet := os.LookupEnv("GOPHER_PROTOCOL")
	testpath := filepath.Join(configPath, "conf", robotConfigFileName)
	_, err = os.Stat(testpath)
	unconfigured := false
	// If custom/conf/robot.yaml doesn't exist, look for repository
	// environment variable.
	if err != nil {
		_, ok := os.LookupEnv("GOPHER_CUSTOM_REPOSITORY")
		// This is true only when creating a new robot
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
		// TODO: shouldn't be needed, remove if no strange errors.
		// os.Unsetenv("GOPHER_UNCONFIGURED")
		if !protoSet {
			termStart()
		}
	}

	// Set up Logging
	var logger *log.Logger
	logOut := os.Stdout
	if len(logFile) == 0 {
		logFile = os.Getenv("GOPHER_LOGFILE")
	}
	if len(logFile) == 0 && cliOp {
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

	if !cliOp {
		if unconfigured {
			Log(robot.Info, "Gopherbot starting unconfigured; no robot.yaml/gopherbot.yaml found")
		} else if ideMode {
			Log(robot.Info, "Gopherbot starting in IDE mode (GOPHER_IDE set); running in $HOME using 'mem' brain and 'terminal' connector")
		} else {
			Log(robot.Info, "Gopherbot starting up!")
		}
		lle := os.Getenv("GOPHER_LOGLEVEL")
		if len(lle) > 0 {
			loglevel := logStrToLevel(lle)
			setLogLevel(loglevel)
		}

		if penvErr != nil {
			Log(robot.Info, "No private environment loaded from '.env': %v\n", penvErr)
		} else {
			Log(robot.Info, "Loaded initial private environment from '%s'\n", envFile)
		}
		if len(fixed) > 0 {
			Log(robot.Warn, "Notice! Fixed invalid file modes for environment file(s): %s", strings.Join(fixed, ", "))
		}

		// Create the 'bot and load configuration, supplying configpath and installpath.
		// When loading configuration, gopherbot first loads default configuration
		// from internal config, then loads from configpath/conf/..., which
		// overrides defaults.
		logger.Printf("Starting up with config dir: %s, and install dir: %s\n", configPath, installPath)
		checkprivsep()
	}

	// Process CLI commands that don't need/want full initBot + brain
	switch cliCommand {
	case "dump", "validate":
		processCLI(usage)
		os.Exit(0)
	}

	initBot()

	if cliOp {
		go runBrain()
		processCLI(usage)
		brainQuit()
		os.Exit(0)
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
	Log(robot.Info, "robot quit/exit/restart")
	time.Sleep(time.Second)
	if restart {
		raisePrivPermanent("restart is set, re-exec'ing")
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
