package dynamobrain

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterSimpleBrain("dynamo", provider)
}
