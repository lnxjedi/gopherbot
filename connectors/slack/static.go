package slack

import (
	"github.com/lnxjedi/gopherbot/robot"
)

func init() {
	robot.RegisterPlugin("slackutil", slackplugin)
	robot.RegisterConnector("slack", Initialize)
}
