// +build module

package knock

import "github.com/lnxjedi/gopherbot/robot"

var knockspec = robot.PluginSpec{
	Name:    "knock",
	Handler: knockhandler,
}

var manifest = robot.Manifest{
	Plugins: []robot.PluginSpec{
		knockspec,
	},
}

// GetMa   nifest returns all the handlers available in this plugin
func GetManifest() robot.Manifest {
	return manifest
}
