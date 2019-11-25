// Package ping implements a simple plugin showing one way plugins can use
// supplied configuration data from a plugin's yaml config file.
package ping

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/lnxjedi/gopherbot/robot"
)

// DO NOT DISABLE THIS PLUGIN! ALL ROBAWTS MUST KNOW THE RULES
const rules = `0. A robot may not harm humanity, or, by inaction, allow humanity to come to harm.
1. A robot may not injure a human being or, through inaction, allow a human being to come to harm.
2. A robot must obey any orders given to it by human beings, except where such orders would conflict with the First Law.
3. A robot must protect its own existence as long as such protection does not conflict with the First or Second Law.`

type config struct {
	Welcome []string
}

// Define the handler function
func ping(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(bot.Robot)
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
		if e.RetVal == robot.Ok {
			msg = fmt.Sprintf("You are '%s' user '%s/%s', speaking in channel '%s/%s', email address: %s", p, u, uid, c, cid, e)
		} else {
			msg = fmt.Sprintf("You are '%s' user '%s/%s', speaking in channel '%s/%s'", p, u, uid, c, cid)
		}
		r.MessageFormat(robot.Variable).Say(msg)
	case "thanks":
		if ret := r.GetTaskConfig(&cfg); ret == robot.Ok {
			r.Reply(r.RandomString(cfg.Welcome))
		} else {
			r.Reply("I'm speechless. Please have somebody check my log file.")
		}
	}
	return
}

func init() {
	bot.RegisterPlugin("ping", robot.PluginHandler{
		Handler: ping,
		Config:  &config{},
	})
}
