package bot

import (
	"fmt"
	"log"
	"runtime"
	"syscall"

	"github.com/lnxjedi/gopherbot/robot"
)

var privUID, unprivUID int

/* NOTE on privsep and setuid gopherbot:
Gopherbot "flips" the traditional sense of setuid; gopherbot is normally run
by the desired user, and installed setuid to a non-priviliged account like
"nobody". This makes it possible to run several instances of gopherbot with
different uids on a single host with a single install.
*/

func init() {
	uid := syscall.Getuid()
	euid := syscall.Geteuid()
	if uid != euid {
		privUID = uid
		unprivUID = euid
		runtime.LockOSThread()
		syscall.Setreuid(unprivUID, privUID)
		privSep = true
	}
}

func raiseThreadPriv(reason string) {
	if privSep {
		ruid := syscall.Getuid()
		euid := syscall.Geteuid()
		if euid == privUID {
			tid := syscall.Gettid()
			Log(robot.Debug, "Successful privilege check for '%s'; r/e for thread %d: %d/%d/%d", reason, tid, ruid, euid)
		} else {
			// Not privileged, create a new privileged thread
			runtime.LockOSThread()
			tid := syscall.Gettid()
			err := syscall.Setreuid(unprivUID, privUID)
			if err != nil {
				Log(robot.Error, "Calling Setreuid(%d, %d) in raiseThreadPriv: %v", unprivUID, privUID, err)
				return
			}
			Log(robot.Debug, "Successfully raised privilege for '%s' thread %d; old r/euid %d/%d; new r/euid: %d/%d", reason, tid, ruid, euid, unprivUID, privUID)
		}
	}
}

func dropThreadPriv(reason string) {
	if privSep {
		runtime.LockOSThread()
		tid := syscall.Gettid()
		err := syscall.Setreuid(unprivUID, unprivUID)
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
		ruid := syscall.Getuid()
		euid := syscall.Geteuid()
		tid := syscall.Gettid()
		l.Printf(fmt.Sprintf("Privilege separation initialized; daemon UID %d, unprivileged UID %d; thread %d r/euid: %d/%d\n", privUID, unprivUID, tid, ruid, euid))
	} else {
		l.Printf("Privilege separation not in use\n")
	}
}
