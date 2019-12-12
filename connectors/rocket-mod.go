package main

import (
	"github.com/lnxjedi/gopherbot/connectors/rocket"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetManifest just wraps the function from the module
func GetManifest() robot.Manifest {
	return rocket.GetManifest()
}
