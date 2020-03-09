// +build module

package duo

import "github.com/lnxjedi/robot"

var duospec = robot.PluginSpec{
	Name:    "duo",
	Handler: duohandler,
}

var manifest = robot.Manifest{
	Plugins: []robot.PluginSpec{
		duospec,
	},
}

// GetManifest returns all the handlers available in this plugin
func GetManifest() robot.Manifest {
	return manifest
}
