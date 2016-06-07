// ping implements the most trivial of Go plugins
package ping

import (
	"github.com/parsley42/gopherbot/bot"
)

const defaultConfig = `
# These are used to see if the robot is alive, so should answer in every channel",
AllChannels: true
Help:
- Keywords: [ "ping", "beep" ]
  Helptext: [ "(bot), ping - see if the bot is alive", "(bot), beep - see if the bot can hear you" ]
- Keywords: [ "rules" ]
  Helptext: [ "(bot), what are the rules? - Be sure the robot knows how to conduct his/herself." ]
CommandMatches:
- Command: "ping"
  Regex: "ping"
- Command: "thanks"
  Regex: "(?i:thanks?( you)?!?)"
- Command: "rules"
  Regex: "(?i:(?:what are )?the rules\\??)"
- Command: "hello"
  Regex: "(?i:(?:hi|hello)[.!]?)"
- Command: "beep"
  Regex: "beep"
Config:
  Welcome:
  - "You're welcome!"
  - "Don't mention it"
  - "De nada"
  - "Sure thing"
  - "No problem!"
  - "No problemo!"
  - "Happy to help"
  - "T'was nothing"
`

// DO NOT DISABLE THIS PLUGIN! ALL ROBAWTS MUST KNOW THE RULES
const rules = `0. A robot may not harm humanity, or, by inaction, allow humanity to come to harm.
1. A robot may not injure a human being or, through inaction, allow a human being to come to harm.
2. A robot must obey any orders given to it by human beings, except where such orders would conflict with the First Law.
3. A robot must protect its own existence as long as such protection does not conflict with the First or Second Law.`

type config struct {
	Welcome []string
}

// Define the handler function
func ping(bot bot.Robot, command string, args ...string) {
	// The plugin can handle multiple different commands
	switch command {
	// This isn't really necessary
	case "init":
		// ignore
	case "rules":
		bot.Say(rules)
	case "hello":
		bot.Reply("Howdy. Try 'help' if you want me to do something cool.")
	case "ping":
		bot.Fixed().Reply("PONG")
	case "thanks":
		cfg := config{}
		bot.GetPluginConfig(&cfg)
		bot.Reply(bot.RandomString(cfg.Welcome))
	case "beep":
		if bot.Channel == "" {
			bot.Say("Eh, talking to yourself?")
		} else {
			bot.Say("Did anybody else hear something go \"beep\" ?")
		}
	}
}

func init() {
	bot.RegisterPlugin("ping", bot.PluginHandler{DefaultConfig: defaultConfig, Handler: ping})
}
