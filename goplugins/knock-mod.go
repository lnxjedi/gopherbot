package main

import (
	"github.com/lnxjedi/gopherbot/goplugins/knock"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetManifest just wraps the function from the module
func GetManifest() robot.Manifest {
	return knock.GetManifest()
}
