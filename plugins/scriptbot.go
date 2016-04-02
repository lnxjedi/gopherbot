package scriptbot

import (
	"github.com/parsley42/gobot-chatops"
	"github.com/parsley42/gobot-chatops/slack"
)

func scriptbot(msg *bot.PassiveCmd) (string, error) {
	user := msg.User.GetChatUser()
	bot.Debug("Message: " + msg.Raw)
	bot.Debug("Channel: " + msg.Channel)
	bot.Debug("User nick: " + user.Nick)
	bot.Debug("User realname: " + user.RealName)
	if user.Protocol == "slack" {
		var slacker slack.SlackUser
		slacker = msg.User.(slack.SlackUser)
		slackuser := slacker.GetSlackChatUser()
		bot.Debug("User email: " + slackuser.Email)
	}
	return "", nil
}

func init() {
	bot.Debug("Starting scriptbot plugin")
	bot.RegisterPassiveCommand("scriptbot", scriptbot)
}
