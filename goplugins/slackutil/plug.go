// Package slackutil provides slack-specific utility
package slackutil

import (
	"fmt"
	"regexp"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/nlopes/slack"
)

var idre = regexp.MustCompile(`slack id <@(.*)>`)

// Define the handler function
func slackutil(r *bot.Robot, command string, args ...string) (retval bot.TaskRetVal) {
	switch command {
	// This isn't really necessary
	case "init":
		// ignore
	case "identify":
		if r.Protocol != bot.Slack {
			r.Say("Sorry, that only works with Slack")
			return
		}
		sl := r.Incoming.MessageObject.(*slack.MessageEvent)
		sid := idre.FindStringSubmatch(sl.Text)[1]
		r.Say(fmt.Sprintf("User %s has Slack internal ID %s", args[0], sid))
	}
	return
}

func init() {
	bot.RegisterPlugin("slackutil", bot.PluginHandler{
		Handler: slackutil,
	})
}
