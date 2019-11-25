// +build linux

package bot

import (
	"fmt"
	"log"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lnxjedi/gopherbot/robot"
)

func init() {
	uid := syscall.Getuid()
	euid := syscall.Geteuid()
	if uid != euid {
		privUID = euid
		unprivUID = uid
		runtime.LockOSThread()
		syscall.Syscall(syscall.SYS_SETRESUID, uintptr(euid), uintptr(euid), uintptr(uid))
		privSep = true
	}
}

func privCheck(reason string) {
	if privSep {
		var ruid, euid, suid uintptr
		syscall.Syscall(syscall.SYS_GETRESUID, uintptr(unsafe.Pointer(&ruid)), uintptr(unsafe.Pointer(&euid)), uintptr(unsafe.Pointer(&suid)))
		tid := syscall.Gettid()
		if euid != uintptr(privUID) {
			Log(robot.Error, "Privilege check failed for '%s'; thread %d r/e/suid: %d/%d/%d; e != %d", reason, tid, ruid, euid, suid, privUID)
		} else {
			Log(robot.Debug, "Successful privilege check for '%s'; r/e/suid for thread %d: %d/%d/%d", reason, tid, ruid, euid, suid)
		}
	}
}

// DropThreadPriv exported for use on restart in main()
func DropThreadPriv(reason string) {
	if privSep {
		runtime.LockOSThread()
		var ruid, euid, suid, nruid, neuid, nsuid uintptr
		syscall.Syscall(syscall.SYS_GETRESUID, uintptr(unsafe.Pointer(&ruid)), uintptr(unsafe.Pointer(&euid)), uintptr(unsafe.Pointer(&suid)))
		tid := syscall.Gettid()
		_, _, errno := syscall.Syscall(syscall.SYS_SETRESUID, uintptr(unprivUID), uintptr(unprivUID), uintptr(unprivUID))
		syscall.Syscall(syscall.SYS_GETRESUID, uintptr(unsafe.Pointer(&nruid)), uintptr(unsafe.Pointer(&neuid)), uintptr(unsafe.Pointer(&nsuid)))
		if errno != 0 {
			Log(robot.Error, "Unprivileged setresuid(%d) call failed for '%s': %d; thread %d r/e/suid: %d/%d/%d", privUID, reason, errno, tid, ruid, euid, suid)
		} else {
			Log(robot.Debug, "Dropping privileges for '%s' in thread %d; old r/e/suid: %d/%d/%d, new r/e/suid: %d/%d/%d", reason, tid, ruid, euid, suid, nruid, neuid, nsuid)
		}
	}
}

func checkprivsep(l *log.Logger) {
	if privSep {
		var ruid, euid, suid uintptr
		syscall.Syscall(syscall.SYS_GETRESUID, uintptr(unsafe.Pointer(&ruid)), uintptr(unsafe.Pointer(&euid)), uintptr(unsafe.Pointer(&suid)))
		tid := syscall.Gettid()
		l.Printf(fmt.Sprintf("Privilege separation initialized; daemon UID %d, script UID %d; thread %d r/e/suid: %d/%d/%d\n", privUID, unprivUID, tid, ruid, euid, suid))
	} else {
		l.Printf("Privilege separation not in use\n")
	}
}
