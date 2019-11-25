package main

import (
	"github.com/lnxjedi/gopherbot/goplugins/duo"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetPlugins just wraps the function from the plugin
func GetPlugins() []robot.PluginSpec {
	return duo.GetPlugins()
}
