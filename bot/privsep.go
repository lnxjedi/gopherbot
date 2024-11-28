//go:build linux || dragonfly || freebsd || netbsd || openbsd
// +build linux dragonfly freebsd netbsd openbsd

package bot

import (
	"runtime"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

var (
	privSep                     = false
	privUID, unprivUID, procGID int
	extraGroups                 []uint32
)

/* NOTE on privsep and setuid gopherbot:
Gopherbot "flips" the traditional sense of setuid; gopherbot is normally run
by the desired user, and installed setuid to a non-priviliged account like
"nobody". This makes it possible to run several instances of gopherbot with
different uids on a single host with a single install.
*/

func init() {
	uid := unix.Getuid()
	euid := unix.Geteuid()
	gid := unix.Getgid()
	groups, _ := unix.Getgroups()
	extraGroups = convertIntToUint32(groups)
	if uid != euid {
		privUID = uid
		unprivUID = euid
		procGID = gid
		unix.Umask(0022)
		runtime.LockOSThread()
		unix.Setreuid(unprivUID, privUID)
		privSep = true
	}
}

func convertIntToUint32(intSlice []int) []uint32 {
	uint32Slice := make([]uint32, len(intSlice))
	for i, v := range intSlice {
		uint32Slice[i] = uint32(v)
	}
	return uint32Slice
}

// raisePriv only used to restart
func raisePrivPermanent(reason string) {
	if privSep {
		runtime.LockOSThread()
		tid := unix.Gettid()
		err := unix.Setreuid(privUID, privUID)
		if err != nil {
			ruid := unix.Getuid()
			euid := unix.Geteuid()
			Log(robot.Error, "PRIVSEP - Calling Setreuid(%d, %d) (current r/euid %d/%d) in raisePriv (thread %d): %v", privUID, privUID, ruid, euid, tid, err)
			return
		}
		Log(robot.Debug, "PRIVSEP - Successfully raised privilege permanently for '%s' thread %d; new r/euid: %d/%d", reason, tid, privUID, privUID)
	}
}

func debugPriv(reason string) {
	if privSep {
		ruid := unix.Getuid()
		euid := unix.Geteuid()
		tid := unix.Gettid()
		if euid == privUID {
			Log(robot.Debug, "PRIVSEP - Privilege separation check OK for \"%s\"; daemon UID %d, unprivileged UID %d; thread %d; want r/euid: %d/%d\n", reason, ruid, euid, tid, privUID, unprivUID)
		} else {
			Log(robot.Debug, "PRIVSEP - Privilege separation check FAILED for \"%s\"; daemon UID %d, unprivileged UID %d; thread %d; want r/euid: %d/%d\n", reason, ruid, euid, tid, privUID, unprivUID)
		}
	}
}

func checkprivsep() {
	if privSep {
		runtime.LockOSThread()
		ruid := unix.Getuid()
		euid := unix.Geteuid()
		tid := unix.Gettid()
		Log(robot.Info, "PRIVSEP - Privilege separation initialized; daemon UID %d, unprivileged UID %d; thread %d r/euid: %d/%d\n", privUID, unprivUID, tid, ruid, euid)
	} else {
		Log(robot.Info, "PRIVSEP - Privilege separation not in use\n")
	}
}
