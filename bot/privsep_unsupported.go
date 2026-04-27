//go:build !(linux || dragonfly || freebsd || netbsd || openbsd)

package bot

import "github.com/lnxjedi/gopherbot/robot"

// privSep is always disabled on platforms without the setreuid-based
// thread-scoped privilege separation implementation.
var privSep bool

func raiseThreadPriv(reason string) {}

func raiseThreadPrivExternal(reason string) {}

func dropThreadPriv(reason string) {}

func checkprivsep() {
	Log(robot.Info, "PRIVSEP - Privilege separation not available on this platform")
}
