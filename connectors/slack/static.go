package slack

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPlugin("slackutil", slackplugin)
	bot.RegisterConnector("slack", Initialize)
}
