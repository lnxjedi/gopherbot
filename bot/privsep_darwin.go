//go:build darwin

package bot

import (
	"fmt"
	"os"
	"syscall"

	"github.com/lnxjedi/gopherbot/robot"
)

// privSep indicates whether privilege separation is active.
// It is set to true only if privilege separation is successfully initialized.
var privSep bool

var privUID, unprivUID int
var privGID, unprivGID int

func panicIfDarwinSetuidBinaryTampered(unprivUID, unprivGID int) {
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
	if int(st.Uid) != unprivUID {
		panic("binary could be tampered! setuid executable owner mismatch")
	}
	if int(st.Gid) != unprivGID {
		panic("binary could be tampered! setgid executable group mismatch")
	}
}

func init() {
	uid := syscall.Getuid()
	euid := syscall.Geteuid()
	gid := syscall.Getgid()
	egid := syscall.Getegid()
	if uid != euid {
		privUID = uid
		unprivUID = euid
		privGID = gid
		unprivGID = egid
		panicIfDarwinSetuidBinaryTampered(unprivUID, unprivGID)
		syscall.Umask(0022)

		// Darwin keeps the saved setuid/setgid values when only the effective
		// IDs are swapped back to the invoking user. Children re-exec the same
		// setuid binary and permanently commit before extension code starts.
		if err := syscall.Setregid(-1, privGID); err != nil {
			botStdOutLogger.Printf("PRIVSEP - error setting regid in init: %v", err)
			return
		}
		if err := syscall.Setreuid(-1, privUID); err != nil {
			botStdOutLogger.Printf("PRIVSEP - error setting reuid in init: %v", err)
			return
		}

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
		if mode := info.Mode().Perm(); mode != 0755 {
			if err := os.Chmod(cwd, 0755); err != nil {
				botStdOutLogger.Printf("PRIVSEP - error changing permissions of current working directory '%s' to 0755: %v", cwd, err)
				return
			}
			botStdOutLogger.Printf("PRIVSEP - changed permissions of current working directory '%s' from %o to 0755", cwd, mode)
		}

		privSep = true
	}
}

func commitPrivsepChildRole(role privsepChildRole) error {
	switch role {
	case privsepRolePrivileged:
		if err := syscall.Setregid(privGID, privGID); err != nil {
			return fmt.Errorf("setregid privileged: %w", err)
		}
		if err := syscall.Setreuid(privUID, privUID); err != nil {
			return fmt.Errorf("setreuid privileged: %w", err)
		}
	case privsepRoleUnprivileged:
		if err := syscall.Setegid(unprivGID); err != nil {
			return fmt.Errorf("setegid unprivileged: %w", err)
		}
		if err := syscall.Seteuid(unprivUID); err != nil {
			return fmt.Errorf("seteuid unprivileged: %w", err)
		}
		if err := syscall.Setregid(unprivGID, unprivGID); err != nil {
			return fmt.Errorf("setregid unprivileged: %w", err)
		}
		if err := syscall.Setreuid(unprivUID, unprivUID); err != nil {
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
		UID:    syscall.Getuid(),
		EUID:   syscall.Geteuid(),
		GID:    syscall.Getgid(),
		EGID:   syscall.Getegid(),
		Groups: groups,
	}, nil
}

func checkprivsep() {
	if privSep {
		Log(robot.Info, "PRIVSEP - privilege separation initialized; daemon UID/GID %d/%d, unprivileged UID/GID %d/%d; r/euid: %d/%d", privUID, privGID, unprivUID, unprivGID, syscall.Getuid(), syscall.Geteuid())
	} else {
		Log(robot.Info, "PRIVSEP - Privilege separation not in use")
	}
}
