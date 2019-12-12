// Common symbols needed when being built as a module
// +build module

package terminal

import (
	"github.com/lnxjedi/gopherbot/robot"
)

var manifest = robot.Manifest{
	Connector: robot.ConnectorSpec{
		Name:      "terminal",
		Connector: Initialize,
	},
}

// GetManifest returns all the handlers available in this plugin
func GetManifest() robot.Manifest {
	return manifest
}
