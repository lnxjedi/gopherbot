// +build darwin

package bot

import (
	"log"
)

// privSep funcs a noop on Windows

func raiseThreadPriv(reason string) {
}

func raiseThreadPrivExternal(reason string) {
}

func dropThreadPriv(reason string) {
}

func checkprivsep(l *log.Logger) {
}
