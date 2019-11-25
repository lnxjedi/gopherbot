// +build module

package totp

import "github.com/lnxjedi/gopherbot/robot"

var totpspec = robot.PluginSpec{
	Name:    "totp",
	Handler: totphandler,
}

// GetPlugins is the common exported symbol for loadable go plugins.
func GetPlugins() []robot.PluginSpec {
	return []robot.PluginSpec{
		totpspec,
	}
}
