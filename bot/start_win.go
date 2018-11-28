// +build windows

package bot

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"

	"github.com/joho/godotenv"
	"golang.org/x/sys/windows/svc"
)

var started bool
var startLock sync.Mutex
var isIntSess bool
var finish = make(chan struct{})

func init() {
	hostName = os.Getenv("COMPUTERNAME")
}

// Start gets the robot going; Windows can send this at any time, thus the lock (* AFAIK)
func Start(v VersionInfo) {
	startLock.Lock()
	if started {
		startLock.Unlock()
		return
	}
	started = true
	startLock.Unlock()

	botVersion = v

	const svcName = "gopherbot"
	var err error
	isIntSess, err = svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

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
	var winCommand string
	if isIntSess {
		wusage := "manage Windows service, one of: install, remove, start, stop"
		flag.StringVar(&winCommand, "winsvc", "", wusage)
		flag.StringVar(&winCommand, "w", "", wusage+" (shorthand)")
	}
	flag.Parse()

	if winCommand != "" {
		switch winCommand {
		case "install":
			var args []string
			if configPath != "" {
				args = append(args, "-c", configPath)
			}
			err = installService(svcName, "Gopherbot ChatOps chat bot", args)
		case "remove":
			err = removeService(svcName)
		case "start":
			err = startService(svcName)
		case "stop":
			err = controlService(svcName, svc.Stop, svc.Stopped)
		case "pause":
			err = controlService(svcName, svc.Pause, svc.Paused)
		case "continue":
			err = controlService(svcName, svc.Continue, svc.Running)
		default:
			log.Fatalf("invalid command %s", winCommand)
		}
		if err != nil {
			log.Fatalf("failed to %s %s: %v", winCommand, svcName, err)
		}
		return
	}

	var botLogger *log.Logger
	logOut := os.Stdout
	if len(logFile) == 0 {
		logFile = os.Getenv("GOPHER_LOGFILE")
	}
	if !isIntSess && len(logFile) == 0 {
		logFile = "C:/Windows/Temp/gopherbot-startup.log"
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
	botLogger = log.New(logOut, "", log.LstdFlags)
	botLogger.Println("Initialized logging ...")

	installpath = binDirectory

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
		configpath = "."
	}

	environment := path.Join(configpath, "gopherbot.env")
	envErr := godotenv.Overload(environment)

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
		botLogger.Fatal("No connector registered with name:", botCfg.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	h := handler{}
	conn := initializeConnector(h, log.New(ioutil.Discard, "", 0))
	setConnector(conn)

	if isIntSess {
		// Start the connector's main loop for interactive sessions
		stopped := run()
		// ... and wait for the robot to stop
		<-stopped
	} else {
		// Stop logging to startup log when running as a service
		botLogger.SetOutput(ioutil.Discard)
		// Started as a Windows Service
		runService(svcName)
	}
}
