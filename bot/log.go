package bot

import (
	"fmt"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// Log logs messages whenever the connector log level is
// less than the given level
func Log(l robot.LogLevel, m string, v ...interface{}) bool {

	botLogger.Lock()
	currlevel := botLogger.level
	logger := botLogger.l
	botLogger.Unlock()

	if l >= currlevel || l == robot.Audit {
		prefix := logLevelToStr(l) + ":"
		msg := prefix + " " + m
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}

		if l == robot.Fatal {
			logger.Fatal(msg)
		} else {
			logger.Print(msg)
			tsMsg := fmt.Sprintf("%s %s", time.Now().Format("Jan 2 15:04:05"), msg)
			botLogger.Lock()
			botLogger.buffer[botLogger.buffLine] = tsMsg
			botLogger.buffLine = (botLogger.buffLine + 1) % (buffLines - 1)
			botLogger.Unlock()
		}
		return true
	}
	return false
}
