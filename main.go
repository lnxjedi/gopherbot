package main

import (
	"bufio"
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

type LogInfo struct {
	LogFile string // Where the bot should log to
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
	var execpath, installdir, localdir string
	var err error

	// Installdir is where the binary, default config, and stock external
	// plugins are.
	if execpath, err = godaemon.GetExecutablePath(); err != nil {
		log.Fatalf("Couldn't get executable path: %v", err)
	}
	if installdir, err = filepath.Abs(filepath.Dir(execpath)); err != nil {
		log.Fatalf("Couldn't determine install path: %v", err)
	}

	// Process command-line flags
	configDir := flag.String("config", "", "path to the local configuration directory")
	logFile := flag.String("log", "", "path to robot's log file")
	var foreground bool
	fusage := "run the robot in the foreground and log to stderr, for testing/debugging"
	flag.BoolVar(&foreground, "foreground", false, fusage)
	flag.BoolVar(&foreground, "f", false, fusage+" (shorthand)")
	flag.Parse()

	// Localdir is where all user-supplied configuration and
	// external plugins are. It should be in $HOME/.gopherbot, /etc/gopherbot,
	// or TODO: supplied on the command line.
	home := os.Getenv("HOME")
	confSearchPath := []string{
		*configDir,
		os.Getenv("GOPHER_LOCALDIR"),
		home + "/.gopherbot",
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

	// Read the config just to extract the LogFile path
	var cf []byte
	if cf, err = ioutil.ReadFile(localdir + "/conf/gopherbot.json"); err != nil {
		log.Fatalf("Couldn't read conf/gopherbot.json in local configuration directory: %s\n", localdir)
	}
	var l LogInfo
	if err := json.Unmarshal(cf, &l); err != nil {
		log.Fatalf("Error unmarshalling \"%s\": %v", localdir+"/conf/gopherbot.json", err)
	}

	var botLogger *log.Logger
	if foreground {
		botLogger = log.New(os.Stderr, "", log.LstdFlags)
	} else { // run as a daemon
		var f *os.File
		if godaemon.Stage() == godaemon.StageParent {
			var (
				lp  string
				err error
			)
			if len(*logFile) != 0 {
				lp = *logFile
			} else if len(l.LogFile) != 0 {
				lp = l.LogFile
			} else {
				lp = "/tmp/gopherbot.log"
			}
			f, err = os.Create(lp)
			if err != nil {
				log.Fatalf("Couldn't create log file: %v", err)
			}
		}
		stdout, stderr, err := godaemon.MakeDaemon(&godaemon.DaemonAttr{
			Files:         []**os.File{&f},
			ProgramName:   "gopherbot",
			CaptureOutput: true,
		})
		// Don't double-timestamp if another package is using the default logger
		log.SetFlags(0)
		botLogger = log.New(f, "", log.LstdFlags)
		if err != nil {
			botLogger.Fatalln("Problem daemonizing")
		}
		botLogger.Println("Gopherbot started in background")
		// Write stderr and stdout to logs
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				botLogger.Printf("stdout: %s\n", scanner.Text())
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				botLogger.Printf("stderr: %s", scanner.Text())
			}
		}()
	}

	// From here on out we're daemonized, unless -f was passed
	os.Setenv("GOPHER_INSTALLDIR", installdir)
	os.Setenv("GOPHER_LOCALDIR", localdir)
	// Create the 'bot and load configuration, suppying configdir and installdir.
	// When loading configuration, gopherbot first loads default configuration
	// frim installdir/conf/..., then loads from localdir/conf/..., which
	// overrides defaults.
	gopherbot, err := bot.New(localdir, installdir, botLogger)
	if err != nil {
		log.Fatal(fmt.Errorf("Error loading initial configuration: %v", err))
	}

	var conn bot.Connector
	var handler bot.Handler = gopherbot

	switch gopherbot.GetConnectorName() {
	case "slack":
		conn = slack.Start(handler)
	default:
		log.Fatal("Unsupported connector:", gopherbot.GetConnectorName())
	}

	// Initialize the robot with a valid connector
	gopherbot.Init(conn)

	// Start the connector's main loop
	conn.Run()
}
