// +build darwin dragonfly freebsd linux netbsd openbsd

package bot

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	// MakeDaemon from VividCortex - thanks!
	"github.com/VividCortex/godaemon"
)

var unixPlugins bool = true

var started bool
var hostName string
var finish = make(chan struct{})

type botInfo struct {
	LogFile, PidFile string // Locations for the bots log file and pid file
}

func init() {
	hostName = os.Getenv("HOSTNAME")
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
	globalLock.Lock()
	if started {
		globalLock.Unlock()
		return
	}
	started = true
	globalLock.Unlock()
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
	var logFile string
	lusage := "path to robot's log file"
	flag.StringVar(&logFile, "log", "", lusage)
	flag.StringVar(&logFile, "l", "", lusage+" (shorthand)")
	var plainlog bool
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "P", false, plusage+" (shorthand)")
	var daemonize bool
	fusage := "run the robot as a background process"
	flag.BoolVar(&daemonize, "daemonize", false, fusage)
	flag.BoolVar(&daemonize, "d", false, fusage+" (shorthand)")
	flag.Parse()

	// Installdir is where the default config and stock external
	// plugins are. Search some likely locations in case installDir
	// wasn't passed on the command line, or Gopherbot isn't installed
	// in one of the usual system locations.
	execpath, err = godaemon.GetExecutablePath()
	if err == nil {
		execdir, _ = filepath.Abs(filepath.Dir(execpath))
	}
	instSearchPath := []string{
		installDir,
		"/opt/gopherbot",
		"/usr/local/share/gopherbot",
		"/usr/share/gopherbot",
	}
	gosearchpath := os.Getenv("GOPATH")
	if len(gosearchpath) > 0 {
		for _, gopath := range strings.Split(gosearchpath, ":") {
			instSearchPath = append(instSearchPath, gopath+"/src/github.com/uva-its/gopherbot")
		}
	}
	home := os.Getenv("HOME")
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
		"/usr/local/etc/gopherbot",
		"/etc/gopherbot",
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

	// Read the config just to extract the LogFile PidFile path
	var cf []byte
	if cf, err = ioutil.ReadFile(localdir + "/conf/gopherbot.yaml"); err != nil {
		log.Fatalf("Couldn't read conf/gopherbot.yaml in local configuration directory: %s\n", localdir)
	}
	var bi botInfo
	if err = yaml.Unmarshal(cf, &bi); err != nil {
		log.Fatalf("Error unmarshalling \"%s\": %v", localdir+"/conf/gopherbot.yaml", err)
	}

	var botLogger *log.Logger
	if daemonize {
		var f *os.File
		if godaemon.Stage() == godaemon.StageParent {
			var lp string
			if len(logFile) != 0 {
				lp = logFile
			} else if len(bi.LogFile) != 0 {
				lp = bi.LogFile
			} else {
				lp = "/tmp/gopherbot.log"
			}
			f, err = os.Create(lp)
			if err != nil {
				log.Fatalf("Couldn't create log file: %v", err)
			}
			log.Printf("Backgrounding and logging to: %s\n", lp)
		}
		_, _, err = godaemon.MakeDaemon(&godaemon.DaemonAttr{
			Files:         []**os.File{&f},
			ProgramName:   "gopherbot",
			CaptureOutput: false,
		})
		// Don't double-timestamp if another package is using the default logger
		log.SetFlags(0)
		if plainlog {
			botLogger = log.New(f, "", 0)
		} else {
			botLogger = log.New(f, "", log.LstdFlags)
		}
		if err != nil {
			botLogger.Fatalf("Problem daemonizing: %v", err)
		}
	} else { // run in the foreground, log to stderr
		if plainlog {
			botLogger = log.New(os.Stderr, "", 0)
		} else {
			botLogger = log.New(os.Stderr, "", log.LstdFlags)
		}
	}

	// From here on out we're daemonized if -d was passed
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

	connectionStarter, ok := connectors[robot.protocol]
	if !ok {
		botLogger.Fatal("No connector registered with name:", robot.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	h := handler{}
	conn = connectionStarter(h, botLogger)

	// Initialize the robot with a valid connector
	botInit(conn)

	// Start the brain loop
	go runBrain()
	// Start the connector's main loop
	conn.Run(finish)
}
