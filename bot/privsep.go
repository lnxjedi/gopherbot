//go:build linux || dragonfly || freebsd || netbsd || openbsd

package bot

import (
	"fmt"
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
var privGID, unprivGID int

/* NOTE on privsep and setuid gopherbot:
Gopherbot "flips" the traditional sense of setuid; gopherbot is normally run
by the desired user, and installed setuid to a non-privileged account like
"nobody". This makes it possible to run several instances of gopherbot with
different UIDs on a single host with a single install.

The parent engine swaps its effective UID/GID back to the invoking user while
preserving the setuid/setgid nobody saved IDs. File-backed extensions then run
in one-shot child processes that permanently commit to either the invoking user
or the unprivileged account before extension code starts.

Some legacy thread-scoped helpers remain for parent-owned privileged operations
and for migration compatibility. Calls that permanently drop or raise a thread
must still use runtime.LockOSThread() and never unlock that thread.

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

func setRegid(rgid, egid int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_SETREGID, uintptr(rgid), uintptr(egid), 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func nobodyAccountIDs() (int, int, error) {
	nobody, err := user.Lookup("nobody")
	if err != nil {
		return -1, -1, err
	}
	uid, err := strconv.Atoi(nobody.Uid)
	if err != nil {
		return -1, -1, err
	}
	gid, err := strconv.Atoi(nobody.Gid)
	if err != nil {
		return -1, -1, err
	}
	return uid, gid, nil
}

func panicIfSetuidBinaryTampered(unprivUID, unprivGID int) {
	nobodyUID, nobodyGID, err := nobodyAccountIDs()
	if err != nil {
		panic("binary could be tampered! unable to resolve nobody uid/gid")
	}
	if unprivUID != nobodyUID {
		return
	}
	if unprivGID != nobodyGID {
		panic("binary could be tampered! expected setgid nobody executable for privsep")
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
	if info.Mode()&os.ModeSetgid == 0 {
		panic("binary could be tampered! expected setgid bit on executable")
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
	if int(st.Gid) != nobodyGID {
		panic("binary could be tampered! setgid executable group mismatch")
	}
}

func init() {
	uid := unix.Getuid()
	euid := unix.Geteuid()
	gid := unix.Getgid()
	egid := unix.Getegid()
	if uid != euid {
		privUID = uid
		unprivUID = euid
		privGID = gid
		unprivGID = egid
		panicIfSetuidBinaryTampered(unprivUID, unprivGID)
		unix.Umask(0022)

		// Keep the parent engine on the invoking identity while preserving the
		// setuid/setgid nobody saved IDs for child process role commits.
		if err := syscall.Setregid(-1, privGID); err != nil {
			botStdOutLogger.Printf("PRIVSEP - error setting regid in init: %v", err)
			return
		}
		if err := syscall.Setreuid(-1, privUID); err != nil {
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
		if err := setRegid(privGID, privGID); err != nil {
			Log(robot.Error, "PRIVSEP - error calling setRegid(%d, %d) from thread %d in raiseThreadPrivExternal for %s: %v", privGID, privGID, tid, reason, err)
			return
		}
		if err := setReuid(privUID, privUID); err != nil {
			Log(robot.Error, "PRIVSEP - error calling setReuid(%d, %d) from thread %d, r/euid: %d/%d in raiseThreadPrivExternal for %s: %v", privUID, privUID, tid, ruid, euid, reason, err)
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
		if err := setRegid(unprivGID, unprivGID); err != nil {
			botStdOutLogger.Printf("PRIVSEP - error calling setRegid(%d, %d) in dropThreadPriv: %v", unprivGID, unprivGID, err)
			return
		}
		if err := setReuid(unprivUID, unprivUID); err != nil {
			botStdOutLogger.Printf("PRIVSEP - error calling setReuid(%d, %d) in dropThreadPriv: %v", unprivUID, unprivUID, err)
			return
		}
		Log(robot.Debug, "PRIVSEP - successfully dropped privileges for '%s' in thread %d; new r/euid: %d/%d", reason, tid, unprivUID, unprivUID)
	}
}

func commitPrivsepChildRole(role privsepChildRole) error {
	runtime.LockOSThread()
	switch role {
	case privsepRolePrivileged:
		if err := setRegid(privGID, privGID); err != nil {
			return fmt.Errorf("setregid privileged: %w", err)
		}
		if err := setReuid(privUID, privUID); err != nil {
			return fmt.Errorf("setreuid privileged: %w", err)
		}
	case privsepRoleUnprivileged:
		if err := setRegid(unprivGID, unprivGID); err != nil {
			return fmt.Errorf("setregid unprivileged: %w", err)
		}
		if err := setReuid(unprivUID, unprivUID); err != nil {
			return fmt.Errorf("setreuid unprivileged: %w", err)
		}
	default:
		return fmt.Errorf("unsupported role %q", role)
	}
	return nil
}

func currentPrivsepIdentityReport() (privsepIdentityReport, error) {
	groups, err := syscall.Getgroups()
	if err != nil {
		return privsepIdentityReport{}, err
	}
	return privsepIdentityReport{
		UID:    unix.Getuid(),
		EUID:   unix.Geteuid(),
		GID:    unix.Getgid(),
		EGID:   unix.Getegid(),
		Groups: groups,
	}, nil
}

// checkprivsep logs the current state of privilege separation.
// It reports whether privilege separation is active and details the UIDs
// associated with the daemon and the current thread.
func checkprivsep() {
	if privSep {
		ruid := unix.Getuid()
		euid := unix.Geteuid()
		tid := unix.Gettid()
		Log(robot.Info, "PRIVSEP - privilege separation initialized; daemon UID/GID %d/%d, unprivileged UID/GID %d/%d; thread %d r/euid: %d/%d", privUID, privGID, unprivUID, unprivGID, tid, ruid, euid)
	} else {
		Log(robot.Info, "PRIVSEP - Privilege separation not in use")
	}
}
