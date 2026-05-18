//go:build !(linux || dragonfly || freebsd || netbsd || openbsd || darwin)

package bot

import "github.com/lnxjedi/gopherbot/robot"

// privSep is always disabled on platforms without privilege separation support.
var privSep bool
var privUID, unprivUID int
var privGID, unprivGID int

func commitPrivsepChildRole(role privsepChildRole) error {
	return nil
}

func currentPrivsepIdentityReport() (privsepIdentityReport, error) {
	return privsepIdentityReport{}, nil
}

func checkprivsep() {
	Log(robot.Info, "PRIVSEP - Privilege separation not available on this platform")
}
