//go:build linux || dragonfly || freebsd || netbsd || openbsd
// +build linux dragonfly freebsd netbsd openbsd

package bot

import (
	"fmt"
	"log"
	"runtime"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

var privUID, unprivUID int

/* NOTE on privsep and setuid gopherbot:
Gopherbot "flips" the traditional sense of setuid; gopherbot is normally run
by the desired user, and installed setuid to a non-priviliged account like
"nobody". This makes it possible to run several instances of gopherbot with
different uids on a single host with a single install.
*/

func init() {
	uid := unix.Getuid()
	euid := unix.Geteuid()
	if uid != euid {
		privUID = uid
		unprivUID = euid
		unix.Umask(0022)
		runtime.LockOSThread()
		unix.Setreuid(unprivUID, privUID)
		privSep = true
	}
}

func raiseThreadPriv(reason string) {
	if privSep {
		ruid := unix.Getuid()
		euid := unix.Geteuid()
		if euid == privUID {
			tid := unix.Gettid()
			Log(robot.Debug, "Successful privilege check for '%s'; r/e for thread %d: %d/%d", reason, tid, ruid, euid)
		} else {
			// Not privileged, create a new privileged thread
			runtime.LockOSThread()
			tid := unix.Gettid()
			err := unix.Setreuid(unprivUID, privUID)
			if err != nil {
				Log(robot.Error, "Calling Setreuid(%d, %d) in raiseThreadPriv: %v", unprivUID, privUID, err)
				return
			}
			Log(robot.Debug, "Successfully raised privilege for '%s' thread %d; old r/euid %d/%d; new r/euid: %d/%d", reason, tid, ruid, euid, unprivUID, privUID)
		}
	}
}

// When raising for external scripts, we need to permanently raise privilege
// to prevent Go from spawning a child thread unprivileged
func raiseThreadPrivExternal(reason string) {
	if privSep {
		runtime.LockOSThread()
		tid := unix.Gettid()
		err := unix.Setreuid(privUID, privUID)
		if err != nil {
			Log(robot.Error, "Calling Setreuid(%d, %d) in raiseThreadPriv: %v", unprivUID, privUID, err)
			return
		}
		Log(robot.Debug, "Successfully raised privilege permanently for '%s' thread %d; new r/euid: %d/%d", reason, tid, privUID, privUID)
	}
}

func dropThreadPriv(reason string) {
	if privSep {
		runtime.LockOSThread()
		tid := unix.Gettid()
		err := unix.Setreuid(unprivUID, unprivUID)
		if err != nil {
			Log(robot.Error, "Calling Setreuid(%d, %d) in dropThreadPriv: %v", unprivUID, unprivUID, err)
			return
		}
		Log(robot.Debug, "Successfully dropped privileges for '%s' in thread %d; new r/euid: %d/%d", reason, tid, unprivUID, unprivUID)
	}
}

func checkprivsep(l *log.Logger) {
	if privSep {
		runtime.LockOSThread()
		ruid := unix.Getuid()
		euid := unix.Geteuid()
		tid := unix.Gettid()
		l.Printf(fmt.Sprintf("Privilege separation initialized; daemon UID %d, unprivileged UID %d; thread %d r/euid: %d/%d\n", privUID, unprivUID, tid, ruid, euid))
	} else {
		l.Printf("Privilege separation not in use\n")
	}
}
