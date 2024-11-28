//go:build linux || dragonfly || freebsd || netbsd || openbsd

package bot

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"

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

func init() {
	uid := unix.Getuid()
	euid := unix.Geteuid()
	gid := unix.Getgid()
	egid := unix.Getegid()

	if euid == 0 {
		botStdOutLogger.Fatalln("PRIVSEP - gopherbot is running with EUID 0 (root), which is not allowed!!!")
		os.Exit(1)
	}

	if uid != euid {
		// First part of privSep - make sure the robot runs as the user
		// who ran it.
		privUID = uid    // Real UID
		unprivUID = euid // Effective UID (unprivileged)
		privGID = gid    // Real GID
		unprivGID = egid // Effective GID (unprivileged)

		unix.Umask(0022)
		runtime.LockOSThread()

		if err := unix.Setregid(unprivGID, privGID); err != nil {
			botStdOutLogger.Printf("PRIVSEP - Error setting GID to unprivileged GID (%d): %v\n", unprivGID, err)
			return
		}

		if err := unix.Setreuid(unprivUID, privUID); err != nil {
			botStdOutLogger.Printf("PRIVSEP - Error setting UID to unprivileged UID (%d): %v\n", unprivUID, err)
			return
		}
		privSep = true
	}
}

func initializePrivsep() {
	uid := unix.Getuid()
	euid := unix.Geteuid()
	gid := unix.Getgid()
	egid := unix.Getegid()
	helperPath = filepath.Join(installPath, "privsep")

	if privSep {
		if _, err := os.Stat("/proc/self/status"); os.IsNotExist(err) {
			Log(robot.Error, "PRIVSEP - /proc/self/status not found, cannot proceed with privilege separation")
			privSep = false
			return
		} else if err != nil {
			Log(robot.Error, "PRIVSEP - error accessing /proc/self/status: %v", err)
			privSep = false
			return
		}

		if _, err := os.Stat(helperPath); os.IsNotExist(err) {
			Log(robot.Error, "PRIVSEP - privsep helper not found at %s", helperPath)
			privSep = false
			return
		} else if err != nil {
			Log(robot.Error, "PRIVSEP - error accessing privsep helper at %s: %v", helperPath, err)
			privSep = false
			return
		}

		stat, err := os.Stat(helperPath)
		if err != nil {
			Log(robot.Error, "PRIVSEP - error stating privsep helper at %s: %v", helperPath, err)
			privSep = false
			return
		}

		sys, ok := stat.Sys().(*syscall.Stat_t)
		if !ok {
			Log(robot.Error, "PRIVSEP - unable to retrieve syscall.Stat_t for %s", helperPath)
			privSep = false
			return
		}

		// Check if the owner UID is 0 (root)
		if sys.Uid != 0 {
			Log(robot.Error, "PRIVSEP - privsep helper at %s is not owned by root (UID 0)", helperPath)
			privSep = false
			return
		}

		// Check if the setuid bit is set
		if sys.Mode&04000 == 0 {
			Log(robot.Error, "PRIVSEP - privsep helper at %s does not have the setuid bit set", helperPath)
			privSep = false
			return
		}

		Log(robot.Info, "PRIVSEP - Privilege separation initialized, running with EUID/GID %d/%d, RUID/GID %d/%d", euid, egid, uid, gid)
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
