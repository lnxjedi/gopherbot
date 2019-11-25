package main

import (
	"log"

	"github.com/lnxjedi/gopherbot/connectors/terminal"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetInitializer() just wraps the function from the connector
func GetInitializer() (string, func(robot.Handler, *log.Logger) robot.Connector) {
	return terminal.GetInitializer()
}
