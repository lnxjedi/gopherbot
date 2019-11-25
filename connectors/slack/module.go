// Common symbols needed when being built as a module
// +build module

package slack

import (
	"log"

	"github.com/lnxjedi/gopherbot/robot"
)

// GetPlugins is the common exported symbol for loadable go plugins.
func GetPlugins() []robot.PluginSpec {
	return []robot.PluginSpec{
		slackspec,
	}
}

// GetInitializer is the common exported symbol for loadable connector modules.
func GetInitializer() (string, func(robot.Handler, *log.Logger) robot.Connector) {
	return "slack", Initialize
}
