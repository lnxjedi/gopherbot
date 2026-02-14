//go:build linux || dragonfly || freebsd || netbsd || openbsd

package bot

import (
	"os"
	"os/user"
	"runtime"
	"strconv"
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

Privilege separation is achieved by performing syscall.Setreuid(...) in an init
func to initialize all the startup threads, then setReuid syscalls on individual
OS threads to confine privilege changes to specific threads. This ensures that
privilege escalation or demotion does not inadvertently affect other threads
within the process.

The key to solving the privsep problem correctly is:
* call syscall.Setreuid(unpriv, priv) in a func init() to initialize ALL the
  original threads to privileged.
* Always use runtime.LockOSThread() before calling
  syscall.Syscall(SETREUID,unpriv,unpriv) (which only affects the *current* thread,
  not all of them), and NEVER calling UnlockOSThread to ensure that the unpriv
  thread never gets reused, and is destroyed when the goroutine finishes.

See: https://pkg.go.dev/runtime#LockOSThread

*/

// setReuid performs the setreuid syscall to change the real and effective user IDs
// of the *CURRENT OS THREAD ONLY*. This confines the privilege changes to the thread
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

func nobodyAccountUID() (int, error) {
	nobody, err := user.Lookup("nobody")
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(nobody.Uid)
}

func panicIfSetuidBinaryTampered(unprivUID int) {
	nobodyUID, err := nobodyAccountUID()
	if err != nil {
		panic("binary could be tampered! unable to resolve nobody uid")
	}
	if unprivUID != nobodyUID {
		return
	}
	execPath, err := os.Executable()
	if err != nil {
		panic("binary could be tampered! unable to resolve executable path")
	}
	info, err := os.Lstat(execPath)
	if err != nil {
		panic("binary could be tampered! unable to stat executable")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		panic("binary could be tampered! executable path is a symlink")
	}
	if !info.Mode().IsRegular() {
		panic("binary could be tampered! executable path is not a regular file")
	}
	if info.Mode()&os.ModeSetuid == 0 {
		panic("binary could be tampered! expected setuid bit on executable")
	}
	if info.Mode().Perm()&0o022 != 0 {
		panic("binary could be tampered! setuid executable is group/world writable")
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		panic("binary could be tampered! unable to verify executable owner")
	}
	if int(st.Uid) != nobodyUID {
		panic("binary could be tampered! setuid executable owner mismatch")
	}
}

func init() {
	uid := unix.Getuid()
	euid := unix.Geteuid()
	if uid != euid {
		privUID = uid
		unprivUID = euid
		panicIfSetuidBinaryTampered(unprivUID)
		unix.Umask(0022)

		// Attempt to set real and effective UIDs using the raw syscall on ALL the startup
		// threads.
		err := syscall.Setreuid(unprivUID, privUID)
		if err != nil {
			botStdOutLogger.Printf("PRIVSEP - error setting reuid in init: %v", err)
			return
		}

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

		// Successfully initialized privilege separation
		privSep = true
	}
}

func raiseThreadPriv(reason string) {
	if privSep {
		ruid := unix.Getuid()
		euid := unix.Geteuid()
		tid := unix.Gettid()
		if euid == privUID {
			Log(robot.Debug, "PRIVSEP - successful privilege check for '%s'; r/e for thread %d: %d/%d", reason, tid, ruid, euid)
		} else {
			tid := unix.Gettid()
			err := setReuid(unprivUID, privUID)
			if err != nil {
				Log(robot.Error, "PRIVSEP - error calling setReuid(%d, %d) from thread %d, r/euid: %d/%d in raiseThreadPriv for %s: %v", unprivUID, privUID, tid, ruid, euid, reason, err)
				return
			}
			// Most of the time, new threads should already have euid == privUID
			Log(robot.Warn, "PRIVSEP - successfully raised privilege for '%s' thread %d; old r/euid %d/%d; new r/euid: %d/%d", reason, tid, ruid, euid, unprivUID, privUID)
		}
	}
}

// raiseThreadPrivExternal permanently raises privilege for external scripts by
// setting both real and effective UIDs to the privileged UID. This prevents Go
// from spawning child threads with unprivileged UIDs.
func raiseThreadPrivExternal(reason string) {
	if privSep {
		ruid := unix.Getuid()
		euid := unix.Geteuid()
		tid := unix.Gettid()
		runtime.LockOSThread()
		err := setReuid(privUID, privUID)
		if err != nil {
			Log(robot.Error, "PRIVSEP - error calling setReuid(%d, %d) from thread %d, r/euid: %d/%d in raiseThreadPrivExternal for %s: %v", tid, ruid, euid, privUID, privUID, reason, err)
			return
		}
		Log(robot.Debug, "PRIVSEP - successfully raised privilege permanently for '%s' thread %d; new r/euid: %d/%d", reason, tid, privUID, privUID)
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
			botStdOutLogger.Printf("PRIVSEP - error calling setReuid(%d, %d) in dropThreadPriv: %v", unprivUID, unprivUID, err)
			return
		}
		Log(robot.Debug, "PRIVSEP - successfully dropped privileges for '%s' in thread %d; new r/euid: %d/%d", reason, tid, unprivUID, unprivUID)
	}
}

// checkprivsep logs the current state of privilege separation.
// It reports whether privilege separation is active and details the UIDs
// associated with the daemon and the current thread.
func checkprivsep() {
	if privSep {
		ruid := unix.Getuid()
		euid := unix.Geteuid()
		tid := unix.Gettid()
		Log(robot.Info, "PRIVSEP - privilege separation initialized; daemon UID %d, unprivileged UID %d; thread %d r/euid: %d/%d", privUID, unprivUID, tid, ruid, euid)
	} else {
		Log(robot.Info, "PRIVSEP - Privilege separation not in use")
	}
}
