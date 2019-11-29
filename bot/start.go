// +build linux darwin dragonfly freebsd netbsd openbsd

package bot

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Information about privilege separation, set in runtasks_linux.go
var privSep = false

func init() {
	hostName = os.Getenv("HOSTNAME")
}

// Start gets the robot going
func Start(v VersionInfo) (restart bool) {
	botVersion = v

	var installpath, configpath string

	// Process command-line flags
	var logFile string
	lusage := "path to robot's log file"
	flag.StringVar(&logFile, "log", "", lusage)
	flag.StringVar(&logFile, "l", "", lusage+" (shorthand)")
	var plainlog bool
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "P", false, plusage+" (shorthand)")
	flag.Parse()

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
	logger.Println("Initialized logging ...")

	installpath = binDirectory

	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatalf("Unable to determine working directory: %v", err)
	}
	configpath = filepath.Join(cwd, "custom")

	if penvErr != nil {
		logger.Printf("No private environment loaded from '.env': %v\n", penvErr)
	} else {
		logger.Printf("Loaded initial private environment from '.env'\n")
	}

	// Create the 'bot and load configuration, supplying configpath and installpath.
	// When loading configuration, gopherbot first loads default configuration
	// from internal config, then loads from configpath/conf/..., which
	// overrides defaults.
	logger.Printf("Starting up with config dir: %s, and install dir: %s\n", configpath, installpath)
	checkprivsep(logger)
	initBot(configpath, installpath, logger)

	initializeConnector, ok := connectors[botCfg.protocol]
	if !ok {
		logger.Fatalf("No connector registered with name: %s", botCfg.protocol)
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
