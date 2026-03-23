package ssh

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterConnector("ssh", Initialize, robot.ConnectorCapabilities{HiddenCommands: true})
}
