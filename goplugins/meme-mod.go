package main

import (
	"github.com/lnxjedi/gopherbot/goplugins/meme"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetPlugins just wraps the function from the plugin
func GetPlugins() []robot.PluginSpec {
	return meme.GetPlugins()
}
