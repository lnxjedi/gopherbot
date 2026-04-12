package googlechat

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterConnector("googlechat", Initialize)
}
