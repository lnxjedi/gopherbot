// +build windows

package bot

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc/eventlog"
)

var eventLog *eventlog.Log

// Log logs messages whenever the connector log level is
// less than the given level
func Log(l LogLevel, m string, v ...interface{}) bool {

	botLogger.Lock()
	currlevel := botLogger.level
	logger := botLogger.l
	botLogger.Unlock()

	if l >= currlevel || l == Audit {
		prefix := logLevelToStr(l) + ":"
		msg := prefix + " " + m
		if len(v) > 0 {
			msg = fmt.Sprintf(msg, v...)
		}

		if l == Fatal {
			if eventLog != nil {
				eventLog.Error(1, "Fatal error: "+msg)
			}
			logger.Fatal(msg)
		} else {
			logger.Print(msg)
			if eventLog != nil {
				switch l {
				case Info, Audit:
					eventLog.Info(1, msg)
				case Warn:
					eventLog.Warning(1, msg)
				case Error:
					eventLog.Error(1, msg)
				}
			}
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
