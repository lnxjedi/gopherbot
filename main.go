package main

import (
	"github.com/uva-its/gopherbot/bot"

	// If re-compiling Gopherbot, you can comment out unused connectors.
	// Select the connector and provide configuration in conf/gopherbot.yaml
	_ "github.com/uva-its/gopherbot/connectors/slack"

	// If re-compiling, you can comment out unused brain implementations.
	// Select the brain to use and provide configuration in conf/gopherbot.yaml
	_ "github.com/uva-its/gopherbot/brains/file"

	// If re-compiling, you can comment out unused elevator implementations,
	// otherwise you can disable them in conf/plugins/<plugin>.json with
	// "Disabled: true"
	_ "github.com/uva-its/gopherbot/goplugins/duo"
	_ "github.com/uva-its/gopherbot/goplugins/totp"

	// If re-compiling, you can select the plugins you want. Otherwise you can disable
	// them in conf/plugins/<plugin>.json with "Disabled: true"
	_ "github.com/uva-its/gopherbot/goplugins/help"
	_ "github.com/uva-its/gopherbot/goplugins/knock"
	_ "github.com/uva-its/gopherbot/goplugins/links"
	_ "github.com/uva-its/gopherbot/goplugins/lists"
	_ "github.com/uva-its/gopherbot/goplugins/meme"
	_ "github.com/uva-its/gopherbot/goplugins/ping"

	// Enable profiling. You can shrink the binary by removing this, but if the
	// robot ever stops responding for any reason, it's handy for getting a
	// dump of all goroutines.
	_ "net/http/pprof"
)

func main() {
	bot.Start()
}
