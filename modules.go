//go:build test
// +build test

package main

// modules.go - blank imports for all the modules that should be compiled in.

import (
	// Profiling
	// _ "net/http/pprof"

	// *** Included Authorizer plugins
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/groups"

	// *** Included Go plugins, of varying quality
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/duo"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/knock"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/links"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/lists"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/meme"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/ping"

	// *** Included Go jobs external to the main engine
	_ "github.com/lnxjedi/gopherbot/v2/gojobs/go-bootstrap"

	// *** Included Go tasks external to the main engine
	_ "github.com/lnxjedi/gopherbot/v2/gotasks/git-command"
	_ "github.com/lnxjedi/gopherbot/v2/gotasks/ssh-agent"
	_ "github.com/lnxjedi/gopherbot/v2/gotasks/ssh-git-helper"

	// *** Default Slack connector
	_ "github.com/lnxjedi/gopherbot/v2/connectors/slack"

	// *** Default file history
	_ "github.com/lnxjedi/gopherbot/v2/history/file"

	// *** A fantastic brain
	_ "github.com/lnxjedi/gopherbot/v2/brains/dynamodb"
)

/* Uncomment under Profiling above to enable profiling. This inflates
the binary when enabled, but if the robot ever stops responding for any
reason, it's handy for getting a dump of all goroutines. Example usage:

$ go tool pprof http://localhost:8888/debug/pprof/goroutine
...
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) list lnxjedi
Total: 11
ROUTINE ======================== github.com/lnxjedi/gopherbot/v2/bot...
*/
