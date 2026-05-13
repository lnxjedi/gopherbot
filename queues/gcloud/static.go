package gcloud

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterQueueProvider("gcloud", Initialize)
}
