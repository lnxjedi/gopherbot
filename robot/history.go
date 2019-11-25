package robot

import "io"

// HistoryLogger is provided by a HistoryProvider for each job / plugin run
// where it's requested
type HistoryLogger interface {
	// Log a line of output; bot should prefix with STDOUT or STDERR
	Log(line string)
	// Start a new log section with a given name and descriptive info
	Section(name, info string)
	// Close a log file and store
	Close()
}

// HistoryProvider is responsible for storing and retrieving job histories
type HistoryProvider interface {
	// NewHistory provides a HistoryLogger for the given tag / index, and
	// cleans up logs older than maxHistories.
	NewHistory(tag string, index, maxHistories int) (HistoryLogger, error)
	// GetHistory gets an io.Reader() for a given history log
	GetHistory(tag string, index int) (io.Reader, error)
	// GetHistoryURL provides a URL for the history file if there is one
	GetHistoryURL(tag string, index int) (URL string, exists bool)
	// MakeHistoryURL publishes a history to a URL and returns the URL
	MakeHistoryURL(tag string, index int) (URL string, exists bool)
}
