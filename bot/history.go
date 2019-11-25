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
	"log"

	"github.com/lnxjedi/gopherbot/robot"
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

// Map of registered history providers
var historyProviders = make(map[string]func(robot.Handler) robot.HistoryProvider)

// RegisterHistoryProvider allows history implementations to register a function
// with a named provider type that returns a HistoryProvider interface.
func RegisterHistoryProvider(name string, provider func(robot.Handler) robot.HistoryProvider) {
	if stopRegistrations {
		return
	}
	if historyProviders[name] != nil {
		log.Fatal("Attempted registration of duplicate history provider name:", name)
	}
	historyProviders[name] = provider
}
