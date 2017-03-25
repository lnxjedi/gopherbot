// +build darwin dragonfly freebsd linux netbsd openbsd

package bot

import (
	"fmt"
	"time"
)

// Log logs messages whenever the connector log level is
// less than the given level
func Log(l LogLevel, v ...interface{}) {

	logLock.Lock()
	currlevel := logLevel
	logLock.Unlock()

	if l >= currlevel {
		prefix := logLevelToStr(l) + ":"
		p := []interface{}{prefix}
		var msg string
		if len(v) == 1 {
			msg = fmt.Sprintln(prefix, v[0])
		} else {
			v = append(p, v...)
			msg = fmt.Sprintln(v...)
		}

		if l == Fatal {
			b.logger.Fatal(msg)
		} else {
			b.logger.Print(msg)
			tsMsg := fmt.Sprintf("%s %s", time.Now().Format("Jan 2 15:04:05"), msg)
			logLock.Lock()
			logBuffer[logLine] = tsMsg
			logLine = (logLine + 1) % (buffLines - 1)
			logLock.Unlock()
		}
	}
}
