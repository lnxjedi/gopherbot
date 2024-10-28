package duo

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterPlugin("duo", duohandler)
}
