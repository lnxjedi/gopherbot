// Common symbols needed when being built as a module
// +build module

package slack

import (
	"github.com/lnxjedi/gopherbot/robot"
)

var manifest = robot.Manifest{
	Plugins: []robot.PluginSpec{
		slackspec,
	},
	Connector: robot.ConnectorSpec{
		Name:      "slack",
		Connector: Initialize,
	},
}

// GetManifest returns all the handlers available in this plugin
func GetManifest() robot.Manifest {
	return manifest
}
