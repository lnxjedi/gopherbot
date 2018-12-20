package bot

/*
	history.go provides the mechanism and methods for storing and retrieving
	job / plugin run histories of stdout/stderr for a given run. Each time
	a job / plugin is initiated by a trigger, scheduled job, or user command,
	a new history file is started if HistoryLogs is != 0 for the job/plugin.
	The history provider will store histories up to some maximum, and return
	that history based on the index.
*/

import (
	"io"
	"log"
)

type historyLog struct {
	LogIndex   int
	CreateTime string
}

type jobHistory struct {
	NextIndex          int
	Histories          []historyLog
	ExtendedNamespaces []string
}

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
	GetHistoryURL(tag string, index int) (string, bool)
}

// Map of registered history providers
var historyProviders = make(map[string]func(Handler) HistoryProvider)

// RegisterHistoryProvider allows history implementations to register a function
// with a named provider type that returns a HistoryProvider interface.
func RegisterHistoryProvider(name string, provider func(Handler) HistoryProvider) {
	if stopRegistrations {
		return
	}
	if historyProviders[name] != nil {
		log.Fatal("Attempted registration of duplicate history provider name:", name)
	}
	historyProviders[name] = provider
}
