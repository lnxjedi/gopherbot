// +build darwin dragonfly freebsd linux netbsd openbsd

package bot

import (
	"fmt"
	"time"
)

// Log logs messages whenever the connector log level is
// less than the given level
func Log(l LogLevel, v ...interface{}) {

	botLogger.Lock()
	currlevel := botLogger.level
	logger := botLogger.l
	botLogger.Unlock()

	if l >= currlevel || l == Audit {
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
			logger.Fatal(msg)
		} else {
			logger.Print(msg)
			tsMsg := fmt.Sprintf("%s %s", time.Now().Format("Jan 2 15:04:05"), msg)
			botLogger.Lock()
			botLogger.buffer[botLogger.buffLine] = tsMsg
			botLogger.buffLine = (botLogger.buffLine + 1) % (buffLines - 1)
			botLogger.Unlock()
		}
	}
}
