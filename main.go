package main

import (
	"github.com/parsley42/gopherbot/bot"

	// If re-compiling Gopherbot, you can comment out unused connectors.
	// Select the connector and provide configuration in conf/gopherbot.conf
	_ "github.com/parsley42/gopherbot/connectors/slack"

	// If re-compiling, you can comment out unused brain implementations.
	// Select the brain to use and provide configuration in conf/gopherbot.conf
	_ "github.com/parsley42/gopherbot/brains/file"

	// If re-compiling, you can select the plugins you want. Otherwise you can disable
	// them in conf/plugins/<plugin>.json with "Disabled": true
	_ "github.com/parsley42/gopherbot/goplugins/help"
	_ "github.com/parsley42/gopherbot/goplugins/knock"
	_ "github.com/parsley42/gopherbot/goplugins/lists"
	_ "github.com/parsley42/gopherbot/goplugins/meme"
	_ "github.com/parsley42/gopherbot/goplugins/ping"
)

func main() {
	bot.Start()
}
