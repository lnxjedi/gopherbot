// +build linux

package bot

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

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
