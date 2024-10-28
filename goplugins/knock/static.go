package knock

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterPlugin("knock", knockhandler)
}
