// +build module

package meme

import "github.com/lnxjedi/gopherbot/robot"

var memespec = robot.PluginSpec{
	Name:    "memes",
	Handler: memehandler,
}

var manifest = robot.Manifest{
	Plugins: []robot.PluginSpec{
		memespec,
	},
}

// GetManifest returns all the handlers available in this plugin
func GetManifest() robot.Manifest {
	return manifest
}
