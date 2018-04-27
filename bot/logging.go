package bot

import (
	"log"
	"strings"
	"sync"
)

// LogLevel for determining when to output a log entry
type LogLevel int

// Definitions of log levels in order from most to least verbose
const (
	Trace LogLevel = iota
	Debug
	Info
	Audit // For plugins to emit auditable events
	Warn
	Error
	Fatal
)

var logToFile bool // is logging to a file?

// Should be ample for the internal circular log
const buffLines = 500
const maxLines = 50 // maximum lines to send in a message

var botLogger = struct {
	l         *log.Logger
	level     LogLevel
	buffer    []string
	buffLine  int
	pageLines int
	buffPages int
	sync.Mutex
}{
	nil,
	Trace,
	make([]string, buffLines),
	0,
	20,
	buffLines / 20,
	sync.Mutex{},
}

func logStrToLevel(l string) LogLevel {
	switch strings.ToLower(l) {
	case "trace":
		return Trace
	case "debug":
		return Debug
	case "info":
		return Info
	case "audit":
		return Audit
	case "warn":
		return Warn
	default:
		return Error
	}
}

func logLevelToStr(l LogLevel) string {
	switch l {
	case Trace:
		return "Trace"
	case Debug:
		return "Debug"
	case Info:
		return "Info"
	case Audit:
		return "Audit"
	case Warn:
		return "Warning"
	case Error:
		return "Error"
	case Fatal:
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
func setLogLevel(l LogLevel) {
	botLogger.Lock()
	botLogger.level = l
	botLogger.Unlock()
}

func getLogLevel() LogLevel {
	botLogger.Lock()
	l := botLogger.level
	botLogger.Unlock()
	return l
}
