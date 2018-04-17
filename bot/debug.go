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
	pluginID, name, user string // the ID and name of the plugin being debugged, user requesting
	verbose              bool   // do we want feedback for every message the user types?
}

var plugDebug = struct {
	p map[string]*debuggingPlug // map of pluginID to the debuggingPlug struct
	u map[string]*debuggingPlug // map of user to the debuggingPlug struct
	sync.RWMutex
}{
	make(map[string]*debuggingPlug),
	make(map[string]*debuggingPlug),
	sync.RWMutex{},
}

// If the debug statement requests verboseonly, then the user will only get the
// message if verbose debugging was requested.
func (r *Robot) debug(pluginID, msg string, verboseonly bool) {
	if len(pluginID) == 0 && len(r.User) == 0 {
		return
	}
	if len(pluginID) == 0 && !verboseonly {
		return
	}
	plugDebug.RLock()
	ppd, _ := plugDebug.p[pluginID]
	upd, _ := plugDebug.u[r.User]
	plugDebug.RUnlock()
	var targetUser, plugName string
	if ppd == nil {
		if upd == nil {
			return
		}
		// If we can't look up by plugin, and users don't match, we never care
		if upd.user != r.User {
			return
		}
		// User has spoken but the plugin wasn't determined yet
		targetUser = upd.user
		plugName = upd.name
	} else {
		if len(pluginID) > 0 && ppd.pluginID != pluginID {
			return // should only be true for help requests, or authorization / elevation plugin actions
		}
		// If we look up by plugin, users don't need to match if verbose is true
		if ppd.user != r.User && !(verboseonly && ppd.verbose) {
			return
		}
		// We know the plugin, and if users don't match it's verbose
		targetUser = ppd.user
		plugName = ppd.name
	}
	ts := time.Now().Format("2006/01/02 03:04:05")
	r.SendUserMessage(targetUser, fmt.Sprintf("%s DEBUG %s: %s", ts, plugName, msg))
}
