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
func Log(l LogLevel, v ...interface{}) {

	logLock.Lock()
	currlevel := logLevel
	logLock.Unlock()

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

		robot.logger.Print(msg)
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
		logLock.Lock()
		logBuffer[logLine] = tsMsg
		logLine = (logLine + 1) % (buffLines - 1)
		logLock.Unlock()
	}
}
