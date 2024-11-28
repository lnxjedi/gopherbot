//go:build linux || dragonfly || freebsd || netbsd || openbsd

package bot

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"syscall"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

// privSep indicates whether privilege separation is active.
// It is set to true only if privilege separation is successfully initialized.
var privSep bool

var privUID, unprivUID int

/* NOTE on privsep and setuid gopherbot:
Gopherbot "flips" the traditional sense of setuid; gopherbot is normally run
by the desired user, and installed setuid to a non-privileged account like
"nobody". This makes it possible to run several instances of gopherbot with
different UIDs on a single host with a single install.

Privilege separation is achieved by performing setreuid syscalls on individual
OS threads to confine privilege changes to specific threads. This ensures that
privilege escalation or demotion does not inadvertently affect other threads
within the process.
*/

// setReuid performs the setreuid syscall to change the real and effective user IDs
// of the current OS thread. This confines the privilege changes to the thread
// locked by runtime.LockOSThread().
func setReuid(ruid, euid int) error {
	// Perform the setreuid syscall using syscall.Syscall with predefined constants.
	// syscall.SYS_SETREUID is the syscall number for setreuid on AMD64 architectures.
	_, _, errno := syscall.Syscall(syscall.SYS_SETREUID, uintptr(ruid), uintptr(euid), 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func init() {
	uid := unix.Getuid()
	euid := unix.Geteuid()
	if uid != euid {
		// Check and ensure the current working directory has permissions 0755
		cwd, err := os.Getwd()
		if err != nil {
			botStdOutLogger.Printf("PRIVSEP - error getting current working directory: %v", err)
			return
		}

		info, err := os.Stat(cwd)
		if err != nil {
			botStdOutLogger.Printf("PRIVSEP - error stating current working directory '%s': %v", cwd, err)
			return
		}

		mode := info.Mode().Perm()
		if mode != 0755 {
			err = os.Chmod(cwd, 0755)
			if err != nil {
				botStdOutLogger.Printf("PRIVSEP - error changing permissions of current working directory '%s' to 0755: %v", cwd, err)
				return
			}
			botStdOutLogger.Printf("PRIVSEP - changed permissions of current working directory '%s' from %o to 0755", cwd, mode)
		}

		privUID = uid
		unprivUID = euid
		unix.Umask(0022)
		runtime.LockOSThread()

		// Attempt to set real and effective UIDs using the raw syscall
		err = setReuid(unprivUID, privUID)
		if err != nil {
			botStdOutLogger.Printf("Error setting reuid in init: %v", err)
			return
		}

		// Successfully initialized privilege separation
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
			err := setReuid(unprivUID, privUID)
			if err != nil {
				botStdOutLogger.Printf("Error calling setReuid(%d, %d) in raiseThreadPriv: %v", unprivUID, privUID, err)
				return
			}
			Log(robot.Debug, "Successfully raised privilege for '%s' thread %d; old r/euid %d/%d; new r/euid: %d/%d", reason, tid, ruid, euid, unprivUID, privUID)
		}
	}
}

// raiseThreadPrivExternal permanently raises privilege for external scripts by
// setting both real and effective UIDs to the privileged UID. This prevents Go
// from spawning child threads with unprivileged UIDs.
func raiseThreadPrivExternal(reason string) {
	if privSep {
		runtime.LockOSThread()
		tid := unix.Gettid()
		err := setReuid(privUID, privUID)
		if err != nil {
			botStdOutLogger.Printf("Error calling setReuid(%d, %d) in raiseThreadPrivExternal: %v", privUID, privUID, err)
			return
		}
		Log(robot.Debug, "Successfully raised privilege permanently for '%s' thread %d; new r/euid: %d/%d", reason, tid, privUID, privUID)
	}
}

// dropThreadPriv drops privileges by setting both real and effective UIDs to the
// unprivileged UID. This confines the privilege drop to the current OS thread.
func dropThreadPriv(reason string) {
	if privSep {
		runtime.LockOSThread()
		tid := unix.Gettid()
		err := setReuid(unprivUID, unprivUID)
		if err != nil {
			botStdOutLogger.Printf("Error calling setReuid(%d, %d) in dropThreadPriv: %v", unprivUID, unprivUID, err)
			return
		}
		Log(robot.Debug, "Successfully dropped privileges for '%s' in thread %d; new r/euid: %d/%d", reason, tid, unprivUID, unprivUID)
	}
}

// checkprivsep logs the current state of privilege separation.
// It reports whether privilege separation is active and details the UIDs
// associated with the daemon and the current thread.
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
