package bot

/* debug.go - Provide support for task/plugin debugging. Sends extra logging
information for a given task with level "Info".
*/

import (
	"fmt"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
)

type debuggingTask struct {
	taskID, name string // the ID and name of the task being debugged
	verbose      bool   // do we want feedback for every message the user types?
}

var taskDebug = struct {
	p map[string]*debuggingTask // map of task.name to the debuggingTask struct
	sync.RWMutex
}{
	make(map[string]*debuggingTask),
	sync.RWMutex{},
}

// If the debug statement requests verboseonly, then the user will only get the
// message if verbose debugging was requested.
func debugT(t interface{}, msg string, verboseonly bool) {
	if t == nil {
		return
	}
	task, _, _ := getTask(t)
	debugTask(task, msg, verboseonly)
}

func debugTask(task *Task, msg string, verboseonly bool) {
	if task == nil {
		return
	}
	taskDebug.RLock()
	ppd, _ := taskDebug.p[task.name]
	taskDebug.RUnlock()
	var plugName string
	if ppd == nil {
		return
	}
	if verboseonly && !ppd.verbose {
		return
	}
	plugName = ppd.name
	debugLog := fmt.Sprintf("DEBUG %s: %s", plugName, msg)
	// Since Format isn't set right away, we always debug with the configured default
	Log(robot.Info, debugLog)
}
