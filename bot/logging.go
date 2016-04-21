package bot

import "log"

// LogLevel for determining when to output a log entry
type LogLevel int

// Definitions of log levels in order from most to least verbose
const (
	Trace LogLevel = iota
	Debug
	Info
	Warn
	Error
)

type BotLogger interface {
	Log(l LogLevel, v ...interface{})
	GetLogLevel() LogLevel
	// SetLogLevel updates the connector log level
	SetLogLevel(l LogLevel)
}

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
			prefix = "Info"
		case Warn:
			prefix = "Warning:"
		case Error:
			prefix = "Error"
		}
		log.Println(prefix, v)
	}
}

// GetLogLevel returns the current log level
func (b *robot) GetLogLevel() LogLevel {
	b.RLock()
	l := b.level
	b.RUnlock()
	return l
}

// SetLogLevel updates the connector log level
func (b *robot) SetLogLevel(l LogLevel) {
	b.Lock()
	b.level = l
	b.Unlock()
}
