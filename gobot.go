package main

import (
	"fmt"
	"log"
	"os"

	"github.com/parsley42/gobot-chatops/bot"
	// Select a connector, put configuration in $GOBOT_CONFIGDIR/gobot.conf
	"github.com/parsley42/gobot-chatops/connectors/slack"
	// Select the plugins you want
	_ "github.com/parsley42/gobot-chatops/plugins/meme"
)

func main() {
	// Create the 'bot and load configuration'
	gobot, err := bot.Create(os.Getenv("GOBOT_CONFIGDIR"))
	if err != nil {
		log.Fatal(fmt.Errorf("Error loading initial configuration: %v", err))
	}

	var conn bot.Connector
	var handler bot.Handler = gobot

	switch gobot.GetConnectorName() {
	case "slack":
		conn = slack.Start(handler)
	default:
		log.Fatal("Unsupported connector:", gobot.GetConnectorName())
	}

	// Initialize the robot with a valid connector
	gobot.Init(conn)

	// Start the connector's main loop
	conn.Run()
}
