// Common symbols needed when being built as a module
// +build module

package dynamobrain

import (
	"github.com/lnxjedi/robot"
)

var manifest = robot.Manifest{
	Brain: robot.BrainSpec{
		Name:  "dynamo",
		Brain: provider,
	},
}

// GetManifest returns all the handlers available in this plugin
func GetManifest() robot.Manifest {
	return manifest
}
