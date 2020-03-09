// Common symbols needed when being built as a module
// +build module

package rocket

import (
	"github.com/lnxjedi/robot"
)

var manifest = robot.Manifest{
	Connector: robot.ConnectorSpec{
		Name:      "rocket",
		Connector: Initialize,
	},
}

// GetManifest returns all the handlers available in this plugin
func GetManifest() robot.Manifest {
	return manifest
}
