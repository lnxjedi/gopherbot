// The ping plugin is a simple plugin showing one way plugins can use
// supplied configuration data from a plugin's yaml config file.
package ping

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/bot"
)

const defaultConfig = `
# These are used to see if the robot is alive, so should answer in every channel
AllChannels: true
Help:
- Keywords: [ "ping" ]
  Helptext: [ "(bot), ping - see if the bot is alive" ]
- Keywords: [ "rules" ]
  Helptext: [ "(bot), what are the rules? - Be sure the robot knows how to conduct his/herself." ]
- Keywords: [ "whoami", "user", "identity", "handle", "username" ]
  Helptext: [ "(bot), whoami - Get the robot to tell you a little bit about yourself." ]
CommandMatchers:
- Command: "ping"
  Regex: "(?i:ping)"
- Command: "thanks"
  Regex: "(?i:thanks?( you)?!?)"
- Command: "rules"
  Regex: "(?i:(?:what are )?the rules\\??)"
- Command: "whoami"
  Regex: "(?i:whoami)"
- Command: "hello"
  Regex: "(?i:(?:hi|hello|howdy)[.!]?)"
# These can be overridden by adding a Config: section to conf/plugins/ping.yaml
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
func ping(r *bot.Robot, command string, args ...string) (retval bot.TaskRetVal) {
	var cfg *config
	// The plugin can handle multiple different commands
	switch command {
	// This isn't really necessary
	case "init":
		// ignore
	case "rules":
		r.Say(rules)
	case "hello":
		r.Reply("Howdy. Try 'help' if you want usage information.")
	case "ping":
		r.Fixed().Reply("PONG")
	case "whoami":
		u := r.User
		uid := r.ProtocolUser
		c := r.Channel
		cid := r.ProtocolChannel
		p := r.Protocol
		e := r.GetSenderAttribute("email")
		var msg string
		if e.RetVal == bot.Ok {
			msg = fmt.Sprintf("You are '%s' user '%s/%s', speaking in channel '%s/%s', email address: %s", p, u, uid, c, cid, e)
		} else {
			msg = fmt.Sprintf("You are '%s' user '%s/%s', speaking in channel '%s/%s'", p, u, uid, c, cid)
		}
		r.MessageFormat(bot.Variable).Say(msg)
	case "thanks":
		if ret := r.GetTaskConfig(&cfg); ret == bot.Ok {
			r.Reply(r.RandomString(cfg.Welcome))
		} else {
			r.Reply("I'm speechless. Please have somebody check my log file.")
		}
	}
	return
}

func init() {
	bot.RegisterPlugin("ping", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       ping,
		Config:        &config{},
	})
}
