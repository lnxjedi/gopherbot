// +build windows

package bot

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/kardianos/osext"
	"golang.org/x/sys/windows/svc"
)

var started bool
var isIntSess bool
var hostName string
var conn Connector
var botLogger *log.Logger
var finish = make(chan struct{})

func init() {
	hostName = os.Getenv("COMPUTERNAME")
}

func dirExists(path string) bool {
	if len(path) == 0 {
		return false
	}
	ds, err := os.Stat(path)
	if err != nil {
		return false
	}
	if ds.Mode().IsDir() {
		return true
	}
	return false
}

// Start gets the robot going
func Start() {
	botLock.Lock()
	if started {
		botLock.Unlock()
		return
	}
	started = true
	botLock.Unlock()

	const svcName = "gopherbot"
	var err error
	isIntSess, err = svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

	var execpath, execdir, installdir, localdir string

	// Process command-line flags
	var configDir string
	cusage := "path to the local configuration directory"
	flag.StringVar(&configDir, "config", "", cusage)
	flag.StringVar(&configDir, "c", "", cusage+" (shorthand)")
	var installDir string
	iusage := "path to the local install directory containing default/stock configuration"
	flag.StringVar(&installDir, "install", "", iusage)
	flag.StringVar(&installDir, "i", "", iusage+" (shorthand)")
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
			if configDir != "" {
				args = append(args, "-c", configDir)
			}
			if installDir != "" {
				args = append(args, "-i", installDir)
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

	if isIntSess {
		botLogger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		if logFile == "" {
			logFile = "C:/Windows/Temp/gopherbot-startup.log"
		}
		f, err := os.Create(logFile)
		if err != nil {
			log.Fatal("Unable to open log file")
		}
		botLogger = log.New(f, "", log.LstdFlags)
	}
	botLogger.Println("Starting up ...")

	// Installdir is where the default config and stock external
	// plugins are. Search some likely locations in case installDir
	// wasn't passed on the command line, or Gopherbot isn't installed
	// in one of the usual system locations.
	execpath, err = osext.ExecutableFolder()
	if err == nil {
		execdir = execpath
	}
	instSearchPath := []string{
		installDir,
		`C:/Program Files`,
		`C:/Program Files (x86)`,
	}
	gosearchpath := os.Getenv("GOPATH")
	if len(gosearchpath) > 0 {
		for _, gopath := range strings.Split(gosearchpath, ":") {
			instSearchPath = append(instSearchPath, gopath+"/src/github.com/uva-its/gopherbot")
		}
	}
	home := os.Getenv("USERPROFILE")
	if len(home) > 0 {
		instSearchPath = append(instSearchPath, home+"/go/src/github.com/uva-its/gopherbot")
	}
	instSearchPath = append(instSearchPath, execdir)
	for _, spath := range instSearchPath {
		if len(spath) > 0 && dirExists(spath+"/lib") {
			installdir = spath
			break
		}
	}
	if len(installdir) == 0 {
		botLogger.Println("Install directory not found, exiting")
		os.Exit(0)
	}

	// Localdir is where all user-supplied configuration and
	// external plugins are.
	confSearchPath := []string{
		configDir,
		`C:/Windows/gopherbot`,
	}
	if len(home) > 0 {
		confSearchPath = append(confSearchPath, home+"/.gopherbot")
	}
	for _, spath := range confSearchPath {
		if len(spath) > 0 && dirExists(spath) {
			localdir = spath
			break
		}
	}
	if len(localdir) == 0 {
		botLogger.Println("Couldn't locate local configuration directory, exiting")
		os.Exit(0)
	}

	// Create the 'bot and load configuration, supplying configdir and installdir.
	// When loading configuration, gopherbot first loads default configuration
	// from internal config, then loads from localdir/conf/..., which
	// overrides defaults.
	os.Setenv("GOPHER_INSTALLDIR", installdir)
	os.Setenv("GOPHER_CONFIGDIR", localdir)
	err = newBot(localdir, installdir, botLogger)
	if err != nil {
		botLogger.Fatal(fmt.Errorf("Error loading initial configuration: %v", err))
	}
	botLogger.Printf("Starting up with localdir: %s, and installdir: %s\n", localdir, installdir)

	connectionStarter, ok := connectors[b.protocol]
	if !ok {
		botLogger.Fatal("No connector registered with name:", b.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	h := handler{}
	conn = connectionStarter(h, log.New(ioutil.Discard, "", 0))

	// Initialize the robot with a valid connector
	botInit(conn)

	// Start the brain loop
	go runBrain()
	if isIntSess {
		// Start the connector's main loop for interactive sessions
		conn.Run(finish)
	} else {
		// Stop logging to startup log when running as a service
		b.logger.SetOutput(ioutil.Discard)
		// Started as a Windows Service
		runService(svcName)
	}
}
