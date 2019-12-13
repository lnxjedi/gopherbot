// +build linux darwin dragonfly freebsd netbsd openbsd

package bot

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Information about privilege separation, set in runtasks_linux.go
var privSep = false

// Set for CLI commands
var cliOp = false

func init() {
	hostName = os.Getenv("HOSTNAME")
}

// Start gets the robot going
func Start(v VersionInfo) (restart bool) {
	botVersion = v

	var installpath, configpath string

	// Process command-line flags
	var configPath string
	cusage := "path to the configuration directory"
	flag.StringVar(&configPath, "config", "", cusage)
	flag.StringVar(&configPath, "c", "", "")
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
	run (default) - run the robot
  <command> -h for help on a given command

  Common options:`

	if help {
		fmt.Println(usage)
		flag.PrintDefaults()
		os.Exit(0)
	}

	cliOp = len(flag.Args()) > 0 && flag.Arg(0) != "run"

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
		logOut = lf
	}
	log.SetOutput(logOut)
	logger = log.New(logOut, "", logFlags)
	if !cliOp {
		logger.Println("Initialized logging ...")
	}

	installpath = binDirectory

	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatalf("Unable to determine working directory: %v", err)
	}
	// Configdir is where all user-supplied configuration and
	// external plugins are.
	if len(configPath) != 0 {
		configpath = configPath
	} else if len(envCfgPath) > 0 {
		configpath = envCfgPath
	} else {
		if _, ok := checkDirectory("conf"); ok {
			configpath = cwd
		} else {
			// If not explicitly set or cwd, use "custom" even if it
			// doesn't exist. For compatibility with old installs.
			configpath = filepath.Join(cwd, "custom")
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
		logger.Printf("Starting up with config dir: %s, and install dir: %s\n", configpath, installpath)
		checkprivsep(logger)
	}

	initBot(cwd, configpath, installpath, logger)

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

	// handler{} is just a placeholder struct for implementing the Handler interface
	conn := initializeConnector(handle, logger)

	// NOTE: we use setConnector instead of passing the connector to run()
	// because of the way Windows services were run. Maybe remove eventually?
	setConnector(conn)

	// Start the robot
	stopped := run()
	// ... and wait for the robot to stop
	restart = <-stopped
	return restart
}
