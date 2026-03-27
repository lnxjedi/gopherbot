package cloudflarekv

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterSimpleBrain("cloudflare", provider)
}
