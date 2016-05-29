package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	// MakeDaemon from VividCortex - thanks!
	"github.com/VividCortex/godaemon"

	"github.com/parsley42/gopherbot/bot"

	// If re-compiling Gopherbot, you can comment out unused connectors.
	// Select the connector and provide configuration in $GOPHER_LOCALDIR/conf/gobot.conf
	// TODO: re-implement connectors so they register with names in init()
	"github.com/parsley42/gopherbot/connectors/slack"

	// If re-compiling, you can comment out unused brain implementations.
	// Select the brain to use and provide configuration in $GOPHER_LOCALDIR/conf/gobot.conf
	_ "github.com/parsley42/gopherbot/brains/file"

	// If re-compiling, you can select the plugins you want. Otherwise you can disable
	// them in conf/plugins/<plugin>.json with "Disabled": true
	_ "github.com/parsley42/gopherbot/goplugins/help"
	_ "github.com/parsley42/gopherbot/goplugins/knock"
	_ "github.com/parsley42/gopherbot/goplugins/lists"
	_ "github.com/parsley42/gopherbot/goplugins/meme"
	_ "github.com/parsley42/gopherbot/goplugins/ping"
)

type BotInfo struct {
	LogFile, PidFile string // Locations for the bots log file and pid file
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

func main() {
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
	var pidFile string
	pusage := "path to robot's pid file"
	flag.StringVar(&pidFile, "pid", "", pusage)
	flag.StringVar(&pidFile, "p", "", pusage+" (shorthand)")
	var daemonize bool
	fusage := "run the robot as a background process"
	flag.BoolVar(&daemonize, "daemonize", false, fusage)
	flag.BoolVar(&daemonize, "d", false, fusage+" (shorthand)")
	flag.Parse()

	// Installdir is where the binary, default config, and stock external
	// plugins are.
	if execpath, err = godaemon.GetExecutablePath(); err != nil {
		log.Fatalf("Couldn't get executable path: %v", err)
	}
	if execdir, err = filepath.Abs(filepath.Dir(execpath)); err != nil {
		log.Fatalf("Couldn't determine install path: %v", err)
	}
	instSearchPath := []string{
		installDir,
		os.Getenv("GOPHER_INSTALLDIR"),
		"/usr/local/share/gopherbot",
		"/usr/share/gopherbot",
		execdir,
	}
	for _, spath := range instSearchPath {
		if dirExists(spath) {
			installdir = spath
			break
		}
	}

	// Localdir is where all user-supplied configuration and
	// external plugins are. It should be in $HOME/.gopherbot, /etc/gopherbot,
	// or TODO: supplied on the command line.
	home := os.Getenv("HOME")
	confSearchPath := []string{
		configDir,
		os.Getenv("GOPHER_LOCALDIR"),
		home + "/.gopherbot",
		"/usr/local/etc/gopherbot",
		"/etc/gopherbot",
	}
	for _, spath := range confSearchPath {
		if dirExists(spath) {
			localdir = spath
			break
		}
	}
	if len(localdir) == 0 {
		log.Fatal("Coudln't locate local configuration directory")
	}

	// Read the config just to extract the LogFile PidFile path
	var cf []byte
	if cf, err = ioutil.ReadFile(localdir + "/conf/gopherbot.json"); err != nil {
		log.Fatalf("Couldn't read conf/gopherbot.json in local configuration directory: %s\n", localdir)
	}
	var b BotInfo
	if err := json.Unmarshal(cf, &b); err != nil {
		log.Fatalf("Error unmarshalling \"%s\": %v", localdir+"/conf/gopherbot.json", err)
	}

	var botLogger *log.Logger
	if daemonize {
		var f *os.File
		if godaemon.Stage() == godaemon.StageParent {
			var (
				lp  string
				err error
			)
			if len(logFile) != 0 {
				lp = logFile
			} else if len(b.LogFile) != 0 {
				lp = b.LogFile
			} else {
				lp = "/tmp/gopherbot.log"
			}
			f, err = os.Create(lp)
			if err != nil {
				log.Fatalf("Couldn't create log file: %v", err)
			}
			log.Printf("Backgrounding and logging to: %s\n", lp)
		}
		_, _, err := godaemon.MakeDaemon(&godaemon.DaemonAttr{
			Files:         []**os.File{&f},
			ProgramName:   "gopherbot",
			CaptureOutput: false,
		})
		// Don't double-timestamp if another package is using the default logger
		log.SetFlags(0)
		botLogger = log.New(f, "", log.LstdFlags)
		if err != nil {
			botLogger.Fatalf("Problem daemonizing: %v", err)
		}
		var pf string
		if len(pidFile) != 0 {
			pf = pidFile
		} else if len(b.PidFile) != 0 {
			pf = b.PidFile
		}
		if len(pf) != 0 {
			f, err := os.Create(pf)
			if err != nil {
				botLogger.Printf("Couldn't create pid file: %v", err)
			} else {
				pid := os.Getpid()
				fmt.Fprintf(f, "%d", pid)
				botLogger.Printf("Wrote pid (%d) to: %s\n", pid, pf)
				f.Close()
			}
		}
	} else { // run in the foreground, log to stderr
		botLogger = log.New(os.Stderr, "", log.LstdFlags)
	}
	botLogger.Println("Starting up")

	// From here on out we're daemonized, unless -f was passed
	os.Setenv("GOPHER_INSTALLDIR", installdir)
	os.Setenv("GOPHER_LOCALDIR", localdir)
	// Create the 'bot and load configuration, suppying configdir and installdir.
	// When loading configuration, gopherbot first loads default configuration
	// frim installdir/conf/..., then loads from localdir/conf/..., which
	// overrides defaults.
	gopherbot, err := bot.New(localdir, installdir, botLogger)
	if err != nil {
		botLogger.Fatal(fmt.Errorf("Error loading initial configuration: %v", err))
	}

	var conn bot.Connector
	var handler bot.Handler = gopherbot

	switch gopherbot.GetConnectorName() {
	case "slack":
		conn = slack.Start(handler, botLogger)
	default:
		botLogger.Fatal("Unsupported connector:", gopherbot.GetConnectorName())
	}

	// Initialize the robot with a valid connector
	gopherbot.Init(conn)

	// Start the connector's main loop
	conn.Run()
}
