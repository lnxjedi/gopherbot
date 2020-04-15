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
	"time"

	"github.com/lnxjedi/robot"
)

const histPrefix = "bot:histories:"

// Memory that holds a Ref -> historyLookup record
const histLookup = "bot:histories-lookup"

type historyLog struct {
	LogIndex   int
	Ref        string // 6 hex digits from worker ID
	CreateTime string
	Descriptor string // usually just the branch
}

type historyLookup struct {
	Tag   string
	Index int
}

type pipeHistory struct {
	NextIndex          int
	Histories          []historyLog
	ExtendedNamespaces []string
}

// start a new history log and manage memories
/*
Args:
- tag: pipeline name or job:extended_namespace; newHistory prepends histPrefix
- eid: 8 random hex digits generated in registerActive, for lookups
- descriptor: usually the branch for a repo; differentiates logs for logs
  aggregated with the same log tag, to prevent e.g. entirely separate log
  histories for every feature branch of a build - currently only used
  in ExtendNamespace
- wid: w.id, fallback index when memory fails
- keep: how many of this log to keep
Returns:
- logger: always a history logger, even if it's memory fallback
- url: URL for the log if available
- ref: 8 hex digit reference for e.g. "email|view log abcdef01"
- idx: the run index, or wid fallback - can always be used for tail-log, mail-log in pipeline
*/
func newLogger(tag, eid, descriptor string, wid, keep int) (logger robot.HistoryLogger, url, ref string, idx int) {
	var ph pipeHistory
	// limit the number of logs kept to 64 =< 4096/64 (back of napkin
	// calculation for listing logs w/ max message size of 4096)
	if keep > 64 {
		keep = 64
	}
	// Check out the memory for this specific history
	key := histPrefix + tag
	phtok, _, phret := checkoutDatum(key, &ph, true)
	if phret != robot.Ok {
		Log(robot.Error, "Checking out '%s', no history will be remembered for this pipeline", tag)
		idx = wid
		keep = 0
	} else {
		idx = ph.NextIndex
		ph.NextIndex++
		if ph.NextIndex == maxIndex {
			ph.NextIndex = 0
		}
		// Check out the memory mapping Ref's to logs
		var hmtok string
		var hmret robot.RetVal
		var remove []historyLog
		hm := make(map[string]historyLookup)
		if keep > 0 {
			hmtok, _, hmret = checkoutDatum(histLookup, &hm, true)
			if hmret == robot.Ok {
				ref = eid
				hl := historyLookup{tag, idx}
				hm[ref] = hl
			} else {
				Log(robot.Error, "Checking out '%s' failed for '%s', no lookups will be available for this log", histLookup, tag)
			}
			var start time.Time
			currentCfg.RLock()
			tz := currentCfg.timeZone
			currentCfg.RUnlock()
			if tz != nil {
				start = time.Now().In(tz)
			} else {
				start = time.Now()
			}
			hist := historyLog{
				LogIndex:   idx,
				Ref:        ref,
				Descriptor: descriptor,
				CreateTime: start.Format("Jan 2 15:04:05"),
			}
			ph.Histories = append(ph.Histories, hist)
			l := len(ph.Histories)
			if l > keep {
				remove = ph.Histories[0 : l-keep]
				ph.Histories = ph.Histories[l-keep:]
			}
		}
		mret := updateDatum(key, phtok, ph)
		if mret != robot.Ok {
			Log(robot.Error, "Updating '%s', no history will be remembered for the pipeline", tag)
			idx = wid
			keep = 0
		} else if keep > 0 && hmret == robot.Ok {
			for _, rm := range remove {
				delete(hm, rm.Ref)
			}
			mret := updateDatum(histLookup, hmtok, hm)
			if mret != robot.Ok {
				Log(robot.Error, "Updating '%s' failed for '%s', no lookups will be available for this log", histLookup, tag)
			}
		}
	}
	var err error
	logger, err = interfaces.history.NewLog(tag, idx, keep)
	if err != nil {
		Log(robot.Error, "Starting history for '%s' failed (%v) - falling back to memory log", tag, err)
		idx = wid
		ref = ""
		logger, _ = memHistories.NewLog(tag, idx, 0)
	} else {
		if keep > 0 {
			url, _ = interfaces.history.GetLogURL(tag, idx)
		}
	}
	return
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
