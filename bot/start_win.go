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
)

var started bool
var hostName string

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
	var execpath, execdir, installdir, localdir string
	var err error

	// Process command-line flags
	var configDir string
	cusage := "path to the local configuration directory"
	flag.StringVar(&configDir, "config", "", cusage)
	flag.StringVar(&configDir, "c", "", cusage+" (shorthand)")
	var installDir string
	iusage := "path to the local install directory containing default/stock configuration"
	flag.StringVar(&installDir, "install", "", iusage)
	flag.StringVar(&installDir, "i", "", iusage+" (shorthand)")
	var pidFile string
	pusage := "path to robot's pid file"
	flag.StringVar(&pidFile, "pid", "", pusage)
	flag.StringVar(&pidFile, "p", "", pusage+" (shorthand)")
	var plainlog bool
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "P", false, plusage+" (shorthand)")
	flag.Parse()

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
		log.Println("Install directory not found, exiting")
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
		log.Println("Couldn't locate local configuration directory, exiting")
		os.Exit(0)
	}

	var botLogger *log.Logger
	// later this will check for interactive running vs. running as a service
	if true {
		botLogger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		botLogger.SetOutput(ioutil.Discard)
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
	Log(Info, fmt.Sprintf("Starting up with localdir: %s, and installdir: %s", localdir, installdir))

	var conn Connector

	connectionStarter, ok := connectors[b.protocol]
	if !ok {
		botLogger.Fatal("No connector registered with name:", b.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	h := handler{}
	conn = connectionStarter(h, botLogger)

	// Initialize the robot with a valid connector
	botInit(conn)

	// Start the connector's main loop
	conn.Run()
}
