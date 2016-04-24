// ping implements the most trivial of Go plugins
package ping

import (
	"github.com/parsley42/gopherbot/bot"
)

var (
	gobot   bot.Robot
	botName string
)

// Define the handler function
func ping(bot bot.Robot, channel, user, command string, args ...string) {
	// The plugin can handle multiple different commands
	switch command {
	// This isn't really necessary
	case "init":
		gobot = bot
		botName = user
	case "ping":
		bot.Fixed().Reply("PONG")
	case "beep":
		if channel == "" {
			bot.Say("Eh, talking to yourself?")
		} else {
			bot.Say("Did anybody else hear something go \"beep\" ?")
		}
	}
}

func init() {
	bot.RegisterPlugin("ping", ping)
}
