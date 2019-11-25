package main

import (
	"log"

	"github.com/lnxjedi/gopherbot/connectors/slack"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetPlugins() just wraps the function from the plugin
func GetPlugins() []robot.PluginSpec {
	return slack.GetPlugins()
}

// GetInitializer() just wraps the function from the connector
func GetInitializer() (string, func(robot.Handler, *log.Logger) robot.Connector) {
	return slack.GetInitializer()
}
