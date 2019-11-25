package bot

import (
	"log"
	"strings"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
)

var logToFile bool // is logging to a file?

// Should be ample for the internal circular log
const buffLines = 500
const maxLines = 50 // maximum lines to send in a message

var botLogger = struct {
	l         *log.Logger
	level     robot.LogLevel
	buffer    []string
	buffLine  int
	pageLines int
	buffPages int
	sync.Mutex
}{
	nil,
	robot.Trace,
	make([]string, buffLines),
	0,
	20,
	buffLines / 20,
	sync.Mutex{},
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
