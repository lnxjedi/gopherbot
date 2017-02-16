package bot

import "fmt"

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
func Log(l LogLevel, v ...interface{}) {
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
		}
	}
}

// SetLogLevel updates the connector log level
func setLogLevel(l LogLevel) {
	b.lock.Lock()
	b.level = l
	b.lock.Unlock()
}
