package bot

import (
	"os"
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

// Log logs messages whenever the connector log level is
// less than the given level
func (b *robot) Log(l LogLevel, v ...interface{}) {
	if l >= b.level {
		var prefix string
		switch l {
		case Trace:
			prefix = "Trace:"
		case Debug:
			prefix = "Debug:"
		case Info:
			prefix = "Info:"
		case Warn:
			prefix = "Warning:"
		case Error:
			prefix = "Error:"
		case Fatal:
			prefix = "Fatal:"
		}
		b.logger.Println(prefix, v)
		if l == Fatal {
			time.Sleep(time.Second) // ample time for the log message to reach the file
			os.Exit(1)
		}
	}
}

// SetLogLevel updates the connector log level
func (b *robot) setLogLevel(l LogLevel) {
	b.lock.Lock()
	b.level = l
	b.lock.Unlock()
}
