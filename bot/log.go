package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// initialized in start.go
var botStdErrLogger, botStdOutLogger *log.Logger

// set in start.go
var botStdOutLogging bool

var errorThreshold = robot.Warn

// Log logs messages whenever the connector log level is
// less than the given level
func Log(l robot.LogLevel, m string, v ...interface{}) bool {
	botLogger.Lock()
	currlevel := botLogger.level
	logger := botLogger.l
	botLogger.Unlock()
	prefix := logLevelToStr(l) + ":"
	msg := prefix + " " + m
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	if logger == nil {
		botStdOutLogger.Print(msg)
		return true
	}
	if local && l >= errorThreshold {
		botStdOutLogger.Print(msg)
	}
	if l >= currlevel || l == robot.Audit {
		if l == robot.Fatal {
			logger.Fatal(msg)
		} else {
			if botStdOutLogging && l >= errorThreshold {
				botStdErrLogger.Print(msg)
			} else {
				logger.Print(msg)
			}
			tsMsg := fmt.Sprintf("%s %s\n", time.Now().Format("Jan 2 15:04:05"), msg)
			botLogger.Lock()
			botLogger.buffer[botLogger.buffLine] = tsMsg
			botLogger.buffLine = (botLogger.buffLine + 1) % (buffLines - 1)
			botLogger.Unlock()
		}
		return true
	}
	return false
}
