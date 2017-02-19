package main

import (
	"github.com/uva-its/gopherbot/bot"

	// If re-compiling Gopherbot, you can comment out unused connectors.
	// Select the connector and provide configuration in conf/gopherbot.yaml
	_ "github.com/uva-its/gopherbot/connectors/slack"

	// If re-compiling, you can comment out unused brain implementations.
	// Select the brain to use and provide configuration in conf/gopherbot.yaml
	_ "github.com/uva-its/gopherbot/brains/file"

	// If re-compiling, you can comment out unused elevator implementations.
	// Select the elevator to use and provide configuration in conf/gopherbot.yaml
	_ "github.com/uva-its/gopherbot/elevators/totp"

	// If re-compiling, you can select the plugins you want. Otherwise you can disable
	// them in conf/plugins/<plugin>.json with "Disabled": true
	_ "github.com/uva-its/gopherbot/goplugins/help"
	_ "github.com/uva-its/gopherbot/goplugins/knock"
	_ "github.com/uva-its/gopherbot/goplugins/lists"
	_ "github.com/uva-its/gopherbot/goplugins/meme"
	_ "github.com/uva-its/gopherbot/goplugins/ping"
)

func main() {
	bot.Start()
}
