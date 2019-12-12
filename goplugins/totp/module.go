// +build module

package totp

import "github.com/lnxjedi/gopherbot/robot"

var totpspec = robot.PluginSpec{
	Name:    "totp",
	Handler: totphandler,
}

var manifest = robot.Manifest{
	Plugins: []robot.PluginSpec{
		totpspec,
	},
}

// GetManifest returns all the handlers available in this plugin
func GetManifest() robot.Manifest {
	return manifest
}
