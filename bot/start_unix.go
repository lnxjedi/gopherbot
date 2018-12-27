// +build darwin dragonfly freebsd linux netbsd openbsd

package bot

import (
	"flag"
	"log"
	"os"
	"path"

	"github.com/joho/godotenv"
)

func init() {
	hostName = os.Getenv("HOSTNAME")
}

// Start gets the robot going
func Start(v VersionInfo) {
	botVersion = v

	var installpath, configpath string

	// Process command-line flags
	var configPath string
	cusage := "path to the configuration directory"
	flag.StringVar(&configPath, "config", "", cusage)
	flag.StringVar(&configPath, "c", "", cusage+" (shorthand)")
	var penvPath string
	pepusage := "path to private environment file; defaults to ./.env"
	flag.StringVar(&penvPath, "penv", "", pepusage)
	flag.StringVar(&penvPath, "p", "", pepusage+" (shorthand)")
	var logFile string
	lusage := "path to robot's log file"
	flag.StringVar(&logFile, "log", "", lusage)
	flag.StringVar(&logFile, "l", "", lusage+" (shorthand)")
	var plainlog bool
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "P", false, plusage+" (shorthand)")
	flag.Parse()

	private := ".env"
	if len(penvPath) > 0 {
		private = penvPath
	}
	penvErr := godotenv.Overload(private)
	envCfgPath := os.Getenv("GOPHER_CONFIGDIR")

	// Configdir is where all user-supplied configuration and
	// external plugins are.
	if len(configPath) != 0 {
		configpath = configPath
	} else if len(envCfgPath) > 0 {
		configpath = envCfgPath
	} else {
		if respath, ok := checkDirectory("custom"); ok {
			configpath = respath
		} else {
			configpath = "."
		}
	}

	environment := path.Join(configpath, "gopherbot.env")
	envErr := godotenv.Overload(environment)

	var botLogger *log.Logger
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
		logToFile = true // defined in logging.go
		logOut = lf
	}
	log.SetOutput(logOut)
	botLogger = log.New(logOut, "", logFlags)
	botLogger.Println("Initialized logging ...")

	installpath = binDirectory

	if penvErr != nil {
		botLogger.Printf("No private environment loaded from '%s': %v\n", private, penvErr)
	} else {
		botLogger.Printf("Loaded initial private environment from: %s\n", private)
	}
	if envErr != nil {
		botLogger.Printf("No environment loaded from '%s': %v\n", environment, envErr)
	} else {
		botLogger.Printf("Loaded initial environment from: %s\n", environment)
	}

	// Create the 'bot and load configuration, supplying configpath and installpath.
	// When loading configuration, gopherbot first loads default configuration
	// from internal config, then loads from configpath/conf/..., which
	// overrides defaults.
	lp := "(none)"
	if len(configpath) > 0 {
		lp = configpath
	}
	botLogger.Printf("Starting up with config dir: %s, and install dir: %s\n", lp, installpath)

	initBot(configpath, installpath, botLogger)

	initializeConnector, ok := connectors[botCfg.protocol]
	if !ok {
		botLogger.Fatalf("No connector registered with name: %s", botCfg.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	h := handler{}
	conn := initializeConnector(h, botLogger)

	// NOTE: we use setConnector instead of passing the connector to run()
	// because of the way Windows services run. See 'start_win.go'.
	setConnector(conn)

	// Start the robot
	stopped := run()
	// ... and wait for the robot to stop
	<-stopped
}
