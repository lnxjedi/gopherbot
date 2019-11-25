package main

import (
	"github.com/lnxjedi/gopherbot/goplugins/knock"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetPlugins just wraps the function from the plugin
func GetPlugins() []robot.PluginSpec {
	return knock.GetPlugins()
}
