package main

import (
	dynamobrain "github.com/lnxjedi/gopherbot/brains/dynamodb"
	"github.com/lnxjedi/gopherbot/robot"
)

// GetManifest just wraps the function from the module
func GetManifest() robot.Manifest {
	return dynamobrain.GetManifest()
}
