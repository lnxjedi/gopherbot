package main

import (
	dynamobrain "github.com/lnxjedi/gopherbot/brains/dynamodb"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetBrainProvider just wraps the function from the brain
func GetBrainProvider() (string, func(robot.Handler) robot.SimpleBrain) {
	return dynamobrain.GetBrainProvider()
}
