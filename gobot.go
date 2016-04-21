package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/parsley42/gopherbot/bot"
	// Select a connector, put configuration in $GOPHER_LOCALDIR/gobot.conf
	"github.com/parsley42/gopherbot/connectors/slack"
	// Select the plugins you want
	_ "github.com/parsley42/gopherbot/plugins/meme"
	_ "github.com/parsley42/gopherbot/plugins/ping"
)

func main() {
	var (
		installdir string
		err        error
	)
	installdir = os.Getenv("GOPHER_LOCALDIR")
	installdir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	// Create the 'bot and load configuration, suppying configdir and installdir.
	// When loading configuration, gopherbot first checks GOPHER_LOCALDIR, then
	// installdir/conf
	gopherbot, err := bot.Create(os.Getenv("GOPHER_LOCALDIR"), installdir)
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
