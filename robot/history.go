package robot

import "io"

// HistoryLogger is provided by a HistoryProvider for each job / plugin run
// where it's requested
type HistoryLogger interface {
	// Log a line of output, normally timestamped; bot should prefix with OUT or ERR
	Log(line string)
	// Add a plain line to the log, without a timestamp
	Line(line string)
	// Close a log file against further writes, but keep
	Close()
	// Finalize called after pipeline finishes, log can be removed.
	Finalize()
}

// HistoryProvider is responsible for storing and retrieving job histories
type HistoryProvider interface {
	// NewLog provides a HistoryLogger for the given tag / index, and
	// cleans up logs older than maxHistories.
	NewLog(tag string, index, maxHistories int) (HistoryLogger, error)
	// GetLog gets an io.Reader() for a given history log
	GetLog(tag string, index int) (io.Reader, error)
	// GetLogURL provides a static URL for the history file if there is one
	GetLogURL(tag string, index int) (URL string, exists bool)
	// MakeLogURL publishes a log to a URL and returns the URL; this
	// URL need only be available for a short timespan, e.g. 42 seconds
	MakeLogURL(tag string, index int) (URL string, exists bool)
}
