package bot

/* debug.go - Provide support for plugin debugging. Admin users can use the
'debug' built-in to debug a plugin and get verbose messages sent to them as
a private message detailing everything going on with a plugin.
*/

import (
	"fmt"
	"sync"
	"time"
)

type debuggingPlug struct {
	pluginID, name string // the ID and name of the plugin being debugged
	verbose        bool   // do we want feedback for every message the user types?
}

var plugDebug = struct {
	p map[string]*debuggingPlug // map username to plugin being debugged
	sync.RWMutex
}{
	make(map[string]*debuggingPlug),
	sync.RWMutex{},
}

func (r *Robot) debug(pluginID, msg string, everyMsg bool) {
	if len(r.User) == 0 {
		return
	}
	if len(pluginID) == 0 && !everyMsg {
		return
	}
	plugDebug.RLock()
	up, ok := plugDebug.p[r.User]
	plugDebug.RUnlock()
	if !ok {
		return
	}
	if (pluginID == up.pluginID) || (everyMsg && up.verbose) {
		ts := time.Now().Format("2006/01/02 03:04:05")
		r.SendUserMessage(r.User, fmt.Sprintf("%s DEBUG %s: %s", ts, up.name, msg))
	}
}
