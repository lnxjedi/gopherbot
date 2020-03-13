// +build module

package filehistory

import (
	"github.com/lnxjedi/robot"
)

var fhspec = robot.HistorySpec{
	Name:     "file",
	Provider: provider,
}

func GetManifest() robot.Manifest {
	return robot.Manifest{
		History: fhspec,
	}
}
