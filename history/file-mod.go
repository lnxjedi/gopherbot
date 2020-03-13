// +build module

package main

import (
	"github.com/lnxjedi/gopherbot/history/file"
	"github.com/lnxjedi/robot"
)

// GetManifest just wraps the function from the module
func GetManifest() robot.Manifest {
	return filehistory.GetManifest()
}
