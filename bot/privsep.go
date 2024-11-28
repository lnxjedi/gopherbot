//go:build linux || dragonfly || freebsd || netbsd || openbsd
// +build linux dragonfly freebsd netbsd openbsd

package bot

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

var (
	privSep                                = false
	privUID, unprivUID, privGID, unprivGID int
	helperPath                             string
)

/* NOTE on privsep and setuid gopherbot:
Gopherbot "flips" the traditional sense of setuid; gopherbot is normally run
by the desired user, and installed setuid to a non-priviliged account like
"nobody". This makes it possible to run several instances of gopherbot with
different uids on a single host with a single install.
*/

func initializePrivsep() {
	uid := unix.Getuid()
	euid := unix.Geteuid()
	gid := unix.Getgid()
	egid := unix.Getegid()
	helperPath = filepath.Join(installPath, "privsep")

	if euid == 0 {
		Log(robot.Fatal, "PRIVSEP - gopherbot is running with EUID 0 (root), which is not allowed!!!")
		os.Exit(1)
	}

	if uid != euid {
		if _, err := os.Stat("/proc/self/status"); os.IsNotExist(err) {
			Log(robot.Error, "PRIVSEP - /proc/self/status not found, cannot proceed with privilege separation")
			return
		} else if err != nil {
			Log(robot.Error, "PRIVSEP - error accessing /proc/self/status: %v", err)
			return
		}

		privUID := uid    // Real UID
		unprivUID := euid // Effective UID (unprivileged)
		privGID := gid    // Real GID
		unprivGID := egid // Effective GID (unprivileged)

		unix.Umask(0022)
		runtime.LockOSThread()

		if err := unix.Setregid(unprivGID, privGID); err != nil {
			Log(robot.Error, "Error setting GID to unprivileged GID (%d): %v", unprivGID, err)
			return
		}

		if err := unix.Setreuid(unprivUID, privUID); err != nil {
			Log(robot.Error, "Error setting UID to unprivileged UID (%d): %v", unprivUID, err)
			return
		}

		Log(robot.Info, "Privilege separation initialized, running with EUID/GID %d/%d, RUID/GID %d/%d", euid, egid, uid, gid)
		privSep = true
	}
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
