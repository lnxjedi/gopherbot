package slack

import (
	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/bot"
)

func init() {
	robot.RegisterPlugin("slackutil", slackplugin)
	bot.RegisterConnector("slack", Initialize)
}
