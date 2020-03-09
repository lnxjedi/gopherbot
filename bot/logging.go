package bot

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lnxjedi/robot"
)

// Should be ample for the internal circular log
const buffLines = 500
const maxLines = 50 // maximum lines to send in a message
var logFileName string

type botLoggerInfo struct {
	l         *log.Logger
	f         *os.File
	level     robot.LogLevel
	buffer    []string
	buffLine  int
	pageLines int
	buffPages int
	sync.Mutex
}

var botLogger = botLoggerInfo{
	nil,
	nil,
	robot.Info,
	make([]string, buffLines),
	0,
	20,
	buffLines / 20,
	sync.Mutex{},
}

// note that closing the old output file probably isn't strictly necessary,
// since the old file will be automatically closed by garbage collection.
func (bl *botLoggerInfo) setOutputFile(f *os.File) {
	bl.Lock()
	of := bl.f
	bl.f = f
	bl.Unlock()
	bl.l.SetOutput(f)
	err := of.Close()
	if err != nil {
		Log(robot.Error, "Closing old log file: %v", err)
	}
}

// rename current logfile with given extension and create new log file
func logRotate(extension string) robot.TaskRetVal {
	if len(logFileName) == 0 {
		return robot.Normal
	}
	raiseThreadPriv("rotating log")
	if len(extension) > 0 {
		oldext := filepath.Ext(logFileName)
		barename := strings.TrimSuffix(logFileName, oldext)
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		oldFileName := barename + extension
		os.Remove(oldFileName)
		err := os.Rename(logFileName, oldFileName)
		if err != nil {
			Log(robot.Error, "Renaming '%s' to '%s': %v", logFileName, oldFileName, err)
			return robot.Fail
		}
	} else {
		if err := os.Remove(logFileName); err != nil {
			Log(robot.Error, "Unlinking old log file '%s': %v", logFileName, err)
			return robot.Fail
		}
	}
	lf, err := os.Create(logFileName)
	if err != nil {
		Log(robot.Error, "Creating new log file '%s': %v", logFileName, err)
		return robot.Fail
	}
	botLogger.setOutputFile(lf)
	return robot.Normal
}

func logStrToLevel(l string) robot.LogLevel {
	switch strings.ToLower(l) {
	case "trace":
		return robot.Trace
	case "debug":
		return robot.Debug
	case "info":
		return robot.Info
	case "audit":
		return robot.Audit
	case "warn":
		return robot.Warn
	default:
		return robot.Error
	}
}

func logLevelToStr(l robot.LogLevel) string {
	switch l {
	case robot.Trace:
		return "Trace"
	case robot.Debug:
		return "Debug"
	case robot.Info:
		return "Info"
	case robot.Audit:
		return "Audit"
	case robot.Warn:
		return "Warning"
	case robot.Error:
		return "Error"
	case robot.Fatal:
		return "Fatal"
	default:
		return ""
	}
}

// logPage returns a slice of log strings of length pageLines. If p = 0,
// it returns the most recent page, for p>0 it goes back
func logPage(p int) ([]string, bool) {
	wrapped := false
	botLogger.Lock()
	page := p % botLogger.buffPages
	if page != p {
		wrapped = true
	}
	pageSlice := make([]string, botLogger.pageLines)
	start := (botLogger.buffLine + buffLines - ((page + 1) * botLogger.pageLines))
	start = start - (start/buffLines)*buffLines
	if start+botLogger.pageLines > buffLines {
		copy(pageSlice, botLogger.buffer[start:buffLines])
		copy(pageSlice[buffLines-start:], botLogger.buffer[0:])
	} else {
		copy(pageSlice, botLogger.buffer[start:start+botLogger.pageLines])
	}
	botLogger.Unlock()
	return pageSlice, wrapped
}

// setLogPageLines updates the number of lines per page of log output
func setLogPageLines(l int) int {
	lines := l
	if l > maxLines {
		lines = maxLines
	}
	if l == 0 {
		lines = 1
	}
	botLogger.Lock()
	botLogger.pageLines = lines
	botLogger.buffPages = buffLines / botLogger.pageLines
	botLogger.Unlock()
	return lines
}

// setLogLevel updates the connector log level
func setLogLevel(l robot.LogLevel) {
	botLogger.Lock()
	botLogger.level = l
	botLogger.Unlock()
}

func getLogLevel() robot.LogLevel {
	botLogger.Lock()
	l := botLogger.level
	botLogger.Unlock()
	return l
}
