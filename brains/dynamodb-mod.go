// +build module

package main

import (
	dynamobrain "github.com/lnxjedi/gopherbot/brains/dynamodb"
	"github.com/lnxjedi/robot"
)

// GetManifest just wraps the function from the module
func GetManifest() robot.Manifest {
	return dynamobrain.GetManifest()
}
