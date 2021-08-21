//go:build test
// +build test

package main

import (
	// Profiling
	// _ "net/http/pprof"

	// *** Included Authorizer plugins
	_ "github.com/lnxjedi/gopherbot/goplugins/groups"

	// *** Included Go plugins, of varying quality
	_ "github.com/lnxjedi/gopherbot/goplugins/duo"
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/knock"
	_ "github.com/lnxjedi/gopherbot/goplugins/links"
	_ "github.com/lnxjedi/gopherbot/goplugins/lists"
	_ "github.com/lnxjedi/gopherbot/goplugins/meme"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"
	_ "github.com/lnxjedi/gopherbot/goplugins/totp"

	// *** Default Slack connector
	_ "github.com/lnxjedi/gopherbot/connectors/slack"

	// *** Default file history
	_ "github.com/lnxjedi/gopherbot/history/file"
)

/* Uncomment under Profiling above to enable profiling. This inflates
the binary when enabled, but if the robot ever stops responding for any
reason, it's handy for getting a dump of all goroutines. Example usage:

$ go tool pprof http://localhost:8888/debug/pprof/goroutine
...
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) list lnxjedi
Total: 11
ROUTINE ======================== github.com/lnxjedi/gopherbot/bot...
*/
