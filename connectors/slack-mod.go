// +build module

package main

import (
	"github.com/lnxjedi/gopherbot/connectors/slack"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetManifest just wraps the function from the module
func GetManifest() robot.Manifest {
	return slack.GetManifest()
}
