package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/parsley42/gopherbot/bot"
	// If re-compiling Gopherbot, you can comment out unused connectors.
	// Select the connector and provide configuration in $GOPHER_LOCALDIR/conf/gobot.conf
	// TODO: re-implement connectors so they register with names in init()
	"github.com/parsley42/gopherbot/connectors/slack"
	// If re-compiling, you can comment out unused brain implementations.
	// Select the brain to use and provide configuration in $GOPHER_LOCALDIR/conf/gobot.conf
	_ "github.com/parsley42/gopherbot/brains/file"
	// Select the plugins you want
	_ "github.com/parsley42/gopherbot/goplugins/help"
	_ "github.com/parsley42/gopherbot/goplugins/knock"
	_ "github.com/parsley42/gopherbot/goplugins/meme"
	_ "github.com/parsley42/gopherbot/goplugins/ping"
)

func main() {
	var (
		installdir, localdir string
		err                  error
	)
	// Installdir is where the binary, default config, and stock external
	// plugins are.
	installdir = os.Getenv("GOPHER_INSTALLDIR")
	if len(installdir) == 0 {
		installdir, err = filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}
	}
	os.Setenv("GOPHER_INSTALLDIR", installdir)
	// Localdir is where all user-supplied configuration and
	// external plugins are. The launch script should determine this.
	localdir = os.Getenv("GOPHER_LOCALDIR")
	if len(localdir) == 0 {
		log.Fatal("GOPHER_LOCALDIR not set")
	}

	// Create the 'bot and load configuration, suppying configdir and installdir.
	// When loading configuration, gopherbot first loads default configuration
	// frim installdir/conf/..., then loads from localdir/conf/..., which
	// overrides defaults.
	gopherbot, err := bot.New(localdir, installdir)
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
