// +build darwin dragonfly freebsd netbsd openbsd

package bot

import "log"

// empty declarations for platforms that don't support privsep

func privCheck(reason string) {
}

func dropThreadPriv(reason string) {
}

func checkprivsep(l *log.Logger) {
}
