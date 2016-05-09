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

// DO NOT DISABLE THIS PLUGIN! ALL ROBAWTS MUST KNOW THE RULES
const rules = `0. A robot may not harm humanity, or, by inaction, allow humanity to come to harm.
1. A robot may not injure a human being or, through inaction, allow a human being to come to harm.
2. A robot must obey any orders given to it by human beings, except where such orders would conflict with the First Law.
3. A robot must protect its own existence as long as such protection does not conflict with the First or Second Law.`

// Define the handler function
func ping(bot bot.Robot, command string, args ...string) {
	// The plugin can handle multiple different commands
	switch command {
	// This isn't really necessary
	case "init":
		gobot = bot
		botName = bot.User
	case "rules":
		bot.Say(rules)
	case "hello":
		bot.Reply("Howdy. Try 'help' if you want me to do something cool.")
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
