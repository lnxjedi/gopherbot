// +build linux darwin dragonfly freebsd netbsd openbsd

package bot

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lnxjedi/gopherbot/robot"
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
func Start(v VersionInfo) (restart bool) {
	botVersion = v

	var configpath string

	// Process command-line flags
	var explicitCfgPath string
	cusage := "path to the configuration directory"
	flag.StringVar(&explicitCfgPath, "config", "", cusage)
	flag.StringVar(&explicitCfgPath, "c", "", "")
	var logFile string
	lusage := "path to robot's log file"
	flag.StringVar(&logFile, "log", "", lusage)
	flag.StringVar(&logFile, "l", "", "")
	var plainlog bool
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "p", false, "")
	var help bool
	husage := "help for gopherbot"
	flag.BoolVar(&help, "help", false, husage)
	flag.BoolVar(&help, "h", false, "")
	flag.Parse()

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
	for _, ef := range []string{".env", "private/environment"} {
		if es, err := os.Stat(ef); err == nil {
			em := es.Mode()
			if (uint32(em) & 0066) != 0 {
				log.Fatalf("Invalid file mode '%o' on environment file '%s', aborting", em, ef)
			}
			envFile = ef
		}
	}
	penvErr := godotenv.Overload(envFile)

	envCfgPath := os.Getenv("GOPHER_CONFIGDIR")

	var logger *log.Logger
	logFlags := log.LstdFlags
	if plainlog {
		logFlags = 0
	}
	logOut := os.Stderr
	if len(logFile) == 0 {
		logFile = os.Getenv("GOPHER_LOGFILE")
	}
	if len(logFile) != 0 {
		lf, err := os.Create(logFile)
		if err != nil {
			log.Fatalf("Error creating log file: (%T %v)", err, err)
		}
		fileLog = true
		logOut = lf
	}
	log.SetOutput(logOut)
	logger = log.New(logOut, "", logFlags)
	if !cliOp {
		logger.Println("Initialized logging ...")
	}

	// Configdir is where all user-supplied configuration and
	// external plugins are.
	if len(explicitCfgPath) != 0 {
		configpath = explicitCfgPath
	} else if len(envCfgPath) > 0 {
		configpath = envCfgPath
	} else {
		if _, ok := checkDirectory("conf"); ok {
			configpath = "."
		} else {
			// If not explicitly set or cwd, use "custom" even if it
			// doesn't exist. For compatibility with old installs.
			configpath = "custom"
		}
	}

	if !cliOp {
		if penvErr != nil {
			logger.Printf("No private environment loaded from '.env': %v\n", penvErr)
		} else {
			logger.Printf("Loaded initial private environment from '%s'\n", envFile)
		}

		// Create the 'bot and load configuration, supplying configpath and installpath.
		// When loading configuration, gopherbot first loads default configuration
		// from internal config, then loads from configpath/conf/..., which
		// overrides defaults.
		logger.Printf("Starting up with config dir: %s, and install dir: %s\n", configpath, binDirectory)
		checkprivsep(logger)
	}

	if cliCommand == "dump" {
		botLogger.l = logger
		setLogLevel(robot.Warn)
		if len(flag.Args()) != 3 {
			fmt.Println("DEBUG wrong args")
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

	initBot(configpath, binDirectory, logger)

	if cliOp {
		go runBrain()
		processCLI(usage)
		brainQuit()
		return false
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
	restart = <-done
	raiseThreadPrivExternal("Exiting")
	time.Sleep(time.Second)
	return restart
}
