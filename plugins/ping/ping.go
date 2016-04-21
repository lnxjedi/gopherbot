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
func ping(bot bot.Robot, channel, user, command string, args ...string) error {
	// The plugin can handle multiple different commands
	switch command {
	// This isn't really necessary
	case "start":
		gobot = bot
		botName = user
	case "ping":
		bot.SendChannelMessage(channel, "PONG")
	case "beep":
		bot.SendChannelMessage(channel, "Did anybody else here that beep?")
	}
	return nil
}

func init() {
	bot.RegisterPlugin("ping", ping)
}
