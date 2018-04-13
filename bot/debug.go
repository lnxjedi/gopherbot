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

var plugDebug = struct {
	p map[string]string // map username -> pluginID
	v map[string]bool   // username -> verbose?
	sync.RWMutex
}{
	make(map[string]string),
	make(map[string]bool),
	sync.RWMutex{},
}

func debug(user, pluginID, msg string, everyMsg bool) {
	plugDebug.RLock()
	up, ok := plugDebug.p[user]
	verbose, _ := plugDebug.v[user]
	plugDebug.RUnlock()
	r := &Robot{user, "", Variable, pluginID}
	if (ok && pluginID == up) || (everyMsg && verbose) {
		ts := time.Now().Format("2006/01/02 03:04:05")
		r.SendUserMessage(r.User, fmt.Sprintf("%s DEBUG: %s", ts, msg))
	}
}
