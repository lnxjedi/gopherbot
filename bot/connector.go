package gobot

/* Definition of Connector interface, plus types and constants needed
by a Connector */

type LogLevel int

const (
	Trace LogLevel = iota
	Debug
	Info
	Warn
	Error
)

// type Connector is an interface that all protocols must implement
type Connector interface {
	// JoinChannel joins a channel given it's human-readable name, e.g. "general"
	JoinChannel(c string)
	// SendChannelMessage sends a message to a channel
	SendChannelMessage(chanid string, msg string)
	// SetLogLevel updates the connector log level
	SetLogLevel(l LogLevel)
}
