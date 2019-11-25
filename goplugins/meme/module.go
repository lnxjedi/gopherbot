// +build module

package meme

import "github.com/lnxjedi/gopherbot/robot"

var memespec = robot.PluginSpec{
	Name:    "memes",
	Handler: memehandler,
}

// GetPlugins is the common exported symbol for loadable go plugins.
func GetPlugins() []robot.PluginSpec {
	return []robot.PluginSpec{
		memespec,
	}
}
