// Only needed when built as part of the gopherbot binary
// +build !module

package slack

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("connectors/slack.so")
	bot.RegisterPlugin("slackutil", slackplugin)
	bot.RegisterConnector("slack", Initialize)
}
