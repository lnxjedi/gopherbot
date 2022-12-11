package bot

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// loggers of last resort, initialize early and update in start.go
func init() {
	botStdErrLogger = log.New(os.Stderr, "", log.LstdFlags)
	botStdOutLogger = log.New(os.Stdout, "", log.LstdFlags)
}

// initialized in start.go
var botStdErrLogger, botStdOutLogger *log.Logger

// Set by terminal connector
var terminalWriter io.Writer

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
	// Note logger is nil very briefly on startup
	if logger == nil && l >= currlevel {
		botStdOutLogger.Print(msg)
		return true
	}
	if nullConn && l >= errorThreshold {
		botStdOutLogger.Print(msg)
	}
	if l >= currlevel || l == robot.Audit {
		if l == robot.Fatal {
			logger.Fatal(msg)
		} else {
			if localTerm {
				if terminalWriter != nil {
					terminalWriter.Write([]byte("LOG " + msg + "\n"))
				} else {
					botStdOutLogger.Print("LOG " + msg)
				}
			}
			logger.Print(msg)
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
