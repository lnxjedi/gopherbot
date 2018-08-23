package bot

/* debug.go - Provide support for plugin debugging. Admin users can use the
'debug' built-in to debug a plugin and get verbose messages sent to them as
a private message detailing everything going on with a plugin. Works well with
the 'terminal' connector.
*/

import (
	"fmt"
	"sync"
	"time"
)

type debuggingTask struct {
	taskID, name, user string // the ID and name of the plugin being debugged, user requesting
	verbose            bool   // do we want feedback for every message the user types?
}

var taskDebug = struct {
	p map[string]*debuggingTask // map of taskID to the debuggingTask struct
	u map[string]*debuggingTask // map of user to the debuggingTask struct
	sync.RWMutex
}{
	make(map[string]*debuggingTask),
	make(map[string]*debuggingTask),
	sync.RWMutex{},
}

func (r *botContext) debug(msg string, verboseonly bool) {
	r.debugT(r.currentTask, msg, verboseonly)
}

// If the debug statement requests verboseonly, then the user will only get the
// message if verbose debugging was requested.
func (c *botContext) debugT(t interface{}, msg string, verboseonly bool) {
	if t == nil {
		c.debugTask(nil, msg, verboseonly)
	} else {
		task, _, _ := getTask(t)
		c.debugTask(task, msg, verboseonly)
	}
}

func (c *botContext) debugTask(task *botTask, msg string, verboseonly bool) {
	var taskID string
	if task == nil {
		taskID = ""
	} else {
		taskID = task.taskID
	}
	if len(taskID) == 0 && len(c.User) == 0 {
		return
	}
	if len(taskID) == 0 && !verboseonly {
		return
	}
	taskDebug.RLock()
	ppd, _ := taskDebug.p[taskID]
	upd, _ := taskDebug.u[c.User]
	taskDebug.RUnlock()
	var targetUser, plugName string
	if ppd == nil {
		if upd == nil {
			return
		}
		// Cases where the user is debugging but not the given plugin

		if verboseonly && !upd.verbose {
			return
		}
		// If we can't look up by plugin, and users don't match, we never care
		if upd.user != c.User {
			return
		}
		// We never care about a plugin that's not being debugged
		if len(taskID) > 0 {
			return
		}
		// User has spoken but the plugin wasn't determined yet
		targetUser = upd.user
		plugName = upd.name
	} else {
		// Cases where the given plugin is being debugged, but not necessarily
		// by the user that triggered the debug statement.

		if verboseonly && !ppd.verbose {
			return
		}
		if ppd.user != c.User {
			// If users don't match and verboseonly requested, don't debug
			if verboseonly {
				return
			}
			// If debugging verbose, debug non-verboseonly messages
			if !ppd.verbose {
				return
			}
		}
		if len(taskID) > 0 && ppd.taskID != taskID {
			// should only be true when checking availability for help requests, authorization, or elevation plugins
			return
		}
		// We know the plugin, and if users don't match it's verbose
		targetUser = ppd.user
		plugName = ppd.name
	}
	ts := time.Now().Format("2006/01/02 03:04:05")
	debugLog := fmt.Sprintf("%s DEBUG %s: %s", ts, plugName, msg)
	// Since Format isn't set right away, we always debug with the configured default
	r := c.makeRobot()
	botCfg.RLock()
	r.Format = botCfg.defaultMessageFormat
	botCfg.RUnlock()
	r.SendUserMessage(targetUser, debugLog)
}
