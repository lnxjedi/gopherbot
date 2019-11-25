// +build module

package knock

import "github.com/lnxjedi/gopherbot/robot"

var knockspec = robot.PluginSpec{
	Name:    "knock",
	Handler: knockhandler,
}

// GetPlugins is the common exported symbol for loadable go plugins.
func GetPlugins() []robot.PluginSpec {
	return []robot.PluginSpec{
		knockspec,
	}
}
