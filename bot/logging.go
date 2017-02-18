package bot

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// LogLevel for determining when to output a log entry
type LogLevel int

// Definitions of log levels in order from most to least verbose
const (
	Trace LogLevel = iota
	Debug
	Info
	Warn
	Error
	Fatal
)

// Should be ample for the internal circular log
const buffLines = 500

var logBuffer []string
var logLine int
var pageLines = 20
var buffpages = buffLines / pageLines
var logLock sync.Mutex

func init() {
	logBuffer = make([]string, buffLines)
}

func logStrToLevel(l string) LogLevel {
	switch strings.ToLower(l) {
	case "trace":
		return Trace
	case "debug":
		return Debug
	case "info":
		return Info
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
	case Warn:
		return "Warning"
	case Error:
		return "Error"
	case Fatal:
		return "Fatal"
	}
	return "" // make the compiler happy
}

// Log logs messages whenever the connector log level is
// less than the given level
func Log(l LogLevel, v ...interface{}) {
	if l >= b.level {
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

// logPage returns a slice of log strings of length pageLines. If p = 0,
// it returns the most recent page, for p>0 it goes back
func logPage(p int) ([]string, bool) {
	wrapped := false
	logLock.Lock()
	page := p % buffpages
	if page != p {
		wrapped = true
	}
	pageSlice := make([]string, pageLines)
	start := (logLine + buffLines - ((page + 1) * pageLines))
	start = start - (start/buffLines)*buffLines
	if start+pageLines > buffLines {
		copy(pageSlice, logBuffer[start:buffLines])
		copy(pageSlice[buffLines-start:], logBuffer[0:])
	} else {
		copy(pageSlice, logBuffer[start:start+pageLines])
	}
	logLock.Unlock()
	return pageSlice, wrapped
}

// setLogPageLines updates the number of lines per page of log output
func setLogPageLines(l int) int {
	lines := l
	if l > 100 {
		lines = 100
	}
	logLock.Lock()
	pageLines = lines
	buffpages = buffLines / pageLines
	logLock.Unlock()
	return lines
}

// setLogLevel updates the connector log level
func setLogLevel(l LogLevel) {
	b.lock.Lock()
	b.level = l
	b.lock.Unlock()
}
