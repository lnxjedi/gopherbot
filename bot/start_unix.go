// +build darwin dragonfly freebsd linux netbsd openbsd

package bot

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

var hostName string

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
	var installdir, localdir string
	var err error

	// Process command-line flags
	var configDir string
	cusage := "path to the local configuration directory"
	flag.StringVar(&configDir, "config", "", cusage)
	flag.StringVar(&configDir, "c", "", cusage+" (shorthand)")
	var logFile string
	lusage := "path to robot's log file"
	flag.StringVar(&logFile, "log", "", lusage)
	flag.StringVar(&logFile, "l", "", lusage+" (shorthand)")
	var plainlog bool
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "P", false, plusage+" (shorthand)")
	flag.Parse()

	// Installdir is where the default config and stock external
	// plugins are. Search some likely locations in case installDir
	// wasn't passed on the command line, or Gopherbot isn't installed
	// in one of the usual system locations.
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	installdir, err = filepath.Abs(filepath.Dir(ex))
	if err != nil {
		panic(err)
	}

	// Localdir is where all user-supplied configuration and
	// external plugins are.
	confSearchPath := []string{
		configDir,
		"/usr/local/etc/gopherbot",
		"/etc/gopherbot",
	}
	home := os.Getenv("HOME")
	if len(home) > 0 {
		confSearchPath = append(confSearchPath, home+"/.gopherbot")
	}
	for _, spath := range confSearchPath {
		if len(spath) > 0 && dirExists(spath) {
			localdir = spath
			break
		}
	}

	var botLogger *log.Logger
	logFlags := log.LstdFlags
	if plainlog {
		logFlags = 0
	}
	logOut := os.Stderr
	if logFile != "" {
		lf, err := os.Create(logFile)
		if err != nil {
			log.Fatalf("Error creating log file: (%T %v)", err, err)
		}
		logOut = lf
	}
	log.SetOutput(logOut)
	botLogger = log.New(logOut, "", logFlags)
	botLogger.Println("Initialized logging ...")
	if len(localdir) == 0 {
		botLogger.Println("Couldn't locate local configuration directory, using installed configuration")
	}

	// Create the 'bot and load configuration, supplying configdir and installdir.
	// When loading configuration, gopherbot first loads default configuration
	// from internal config, then loads from localdir/conf/..., which
	// overrides defaults.
	os.Setenv("GOPHER_INSTALLDIR", installdir)
	lp := "(none)"
	if len(localdir) > 0 {
		os.Setenv("GOPHER_CONFIGDIR", localdir)
		lp = localdir
	}
	botLogger.Printf("Starting up with local config dir: %s, and install dir: %s\n", lp, installdir)

	initBot(localdir, installdir, botLogger)

	initializeConnector, ok := connectors[robot.protocol]
	if !ok {
		botLogger.Fatalf("No connector registered with name: %s", robot.protocol)
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
