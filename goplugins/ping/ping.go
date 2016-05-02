// ping implements the most trivial of Go plugins
package ping

import (
	"github.com/parsley42/gopherbot/bot"
)

var (
	gobot   bot.Robot
	botName string
)

var welcome = []string{
	"You're welcome!",
	"Don't mention it",
	"De nada",
	"Sure thing",
	"No problem!",
	"No problemo!",
	"Happy to help",
	"T'was nothing",
}

// Define the handler function
func ping(bot bot.Robot, command string, args ...string) {
	// The plugin can handle multiple different commands
	switch command {
	// This isn't really necessary
	case "init":
		gobot = bot
		botName = bot.User
	case "ping":
		bot.Fixed().Reply("PONG")
	case "thanks":
		bot.Reply(bot.RandomString(welcome))
	case "beep":
		if bot.Channel == "" {
			bot.Say("Eh, talking to yourself?")
		} else {
			bot.Say("Did anybody else hear something go \"beep\" ?")
		}
	}
}

func init() {
	bot.RegisterPlugin("ping", ping)
}
