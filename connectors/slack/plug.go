package slack

import (
	"regexp"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/nlopes/slack"
)

var idre = regexp.MustCompile(`slack id <@(.*)>`)

var slackspec = robot.PluginSpec{
	Name:    "slackutil",
	Handler: slackplugin,
}

// Define the handler function
func slackutil(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	m := r.GetMessage()
	switch command {
	// This isn't really necessary
	case "init":
		// ignore
	case "identify":
		if m.Protocol != robot.Slack {
			r.Say("Sorry, that only works with Slack")
			return
		}
		sl := m.Incoming.MessageObject.(*slack.MessageEvent)
		sid := idre.FindStringSubmatch(sl.Text)[1]
		r.Say("User %s has Slack internal ID %s", args[0], sid)
	}
	return
}

var slackplugin = robot.PluginHandler{
	Handler: slackutil,
}
