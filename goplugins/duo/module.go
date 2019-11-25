// +build module

package duo

import "github.com/lnxjedi/gopherbot/robot"

var duospec = robot.PluginSpec{
	Name:    "duo",
	Handler: duohandler,
}

// GetPlugins is the common exported symbol for loadable go plugins.
func GetPlugins() []robot.PluginSpec {
	return []robot.PluginSpec{
		duospec,
	}
}
