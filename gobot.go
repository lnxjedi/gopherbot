package main

import (
	"fmt"
	"log"
	"os"

	"github.com/parsley42/gobot/bot"
	// Select a connector, put configuration in $GOBOT_CONFIGDIR/gobot.conf
	connector "github.com/parsley42/gobot/connectors/slack"
	// Select the plugins you want
	_ "github.com/parsley42/gobot/plugins/meme"
)

func main() {
	gobot := bot.Create()
	err := gobot.LoadConfig(os.Getenv("GOBOT_CONFIGDIR"))
	if err != nil {
		log.Fatal(fmt.Errorf("Error loading initial configuration: %v", err))
	}

	connector.StartConnector(gobot)
}
