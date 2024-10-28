// Package ping implements a simple plugin showing one way plugins can use
// supplied configuration data from a plugin's yaml config file.
package ping

import (
	"fmt"
	"regexp"

	"github.com/lnxjedi/gopherbot/robot"
)

// DO NOT DISABLE THIS PLUGIN! ALL ROBAWTS MUST KNOW THE RULES
const rules = `0. A robot may not harm humanity, or, by inaction, allow humanity to come to harm.
1. A robot may not injure a human being or, through inaction, allow a human being to come to harm.
2. A robot must obey any orders given to it by human beings, except where such orders would conflict with the First Law.
3. A robot must protect its own existence as long as such protection does not conflict with the First or Second Law.`

type config struct {
	Welcome []string
	Thread  []string
}

var idRegex = regexp.MustCompile(`^<(.*)>$`)

// extractID copied here for simplicity
func extractID(u string) (string, bool) {
	matches := idRegex.FindStringSubmatch(u)
	if len(matches) > 0 {
		return matches[1], true
	}
	return u, false
}

// Define the handler function
func ping(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.GetMessage()
	var cfg *config
	// The plugin can handle multiple different commands
	switch command {
	// This isn't really necessary
	case "init":
		// ignore
	case "rules":
		m.Say(rules)
	case "ignore":
		// uh... ignore!
	case "hello":
		m.Reply("Howdy. Try 'help' if you want usage information.")
	case "ping":
		m.Fixed().Reply("PONG")
	case "thread":
		if ret := m.GetTaskConfig(&cfg); ret == robot.Ok {
			m.ReplyThread(m.RandomString(cfg.Thread))
		} else {
			m.ReplyThread("Sure thing")
		}
	case "whoami":
		u := r.User
		uid, _ := extractID(r.ProtocolUser)
		c := r.Channel
		cid, _ := extractID(r.ProtocolChannel)
		p := r.Protocol
		e := m.GetSenderAttribute("email")
		var msg string
		if e.RetVal == robot.Ok {
			msg = fmt.Sprintf("You are '%s' user '%s/%s', speaking in channel '%s/%s', email address: %s", p, u, uid, c, cid, e)
		} else {
			msg = fmt.Sprintf("You are '%s' user '%s/%s', speaking in channel '%s/%s'", p, u, uid, c, cid)
		}
		m.MessageFormat(robot.Variable).Say(msg)
	case "thanks":
		if ret := m.GetTaskConfig(&cfg); ret == robot.Ok {
			m.Reply(m.RandomString(cfg.Welcome))
		} else {
			m.Reply("I'm speechless. Please have somebody check my log file.")
		}
	}
	return
}

func init() {
	robot.RegisterPlugin("ping", robot.PluginHandler{
		Handler: ping,
		Config:  &config{},
	})
}
