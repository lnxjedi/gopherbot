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

type debuggingPlug struct {
	taskID, name, user string // the ID and name of the plugin being debugged, user requesting
	verbose            bool   // do we want feedback for every message the user types?
}

var plugDebug = struct {
	p map[string]*debuggingPlug // map of taskID to the debuggingPlug struct
	u map[string]*debuggingPlug // map of user to the debuggingPlug struct
	sync.RWMutex
}{
	make(map[string]*debuggingPlug),
	make(map[string]*debuggingPlug),
	sync.RWMutex{},
}

// If the debug statement requests verboseonly, then the user will only get the
// message if verbose debugging was requested.
func (r *botContext) debug(msg string, verboseonly bool) {
	var taskID string
	if r.currentTask == nil {
		taskID = ""
	} else {
		taskID := r.currentTask.taskID
	}
	if len(taskID) == 0 && len(r.User) == 0 {
		return
	}
	if len(taskID) == 0 && !verboseonly {
		return
	}
	plugDebug.RLock()
	ppd, _ := plugDebug.p[taskID]
	upd, _ := plugDebug.u[r.User]
	plugDebug.RUnlock()
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
		if upd.user != r.User {
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
		// Log(Trace, fmt.Sprintf("REMOVE: name: %s, user: %s, verboseonly: %v", ppd.name, r.User, verboseonly))

		if verboseonly && !ppd.verbose {
			return
		}
		if ppd.user != r.User {
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
	robot.RLock()
	r.Format = robot.defaultMessageFormat
	robot.RUnlock()
	r.SendUserMessage(targetUser, debugLog)
	// Log(Debug, debugLog)
}
