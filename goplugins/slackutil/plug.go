// Package slackutil provides slack-specific utility
package slackutil

import (
	"github.com/lnxjedi/gopherbot/bot"
	"github.com/nlopes/slack"
)

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
		r.MessageFormat(bot.Variable).Say(sl.Msg.Text)
	}
	return
}

func init() {
	bot.RegisterPlugin("slackutil", bot.PluginHandler{
		Handler: slackutil,
	})
}
