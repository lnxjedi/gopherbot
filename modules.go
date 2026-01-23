//go:build test
// +build test

package main

// modules.go - blank imports for all the modules that should be compiled in.

import (
	// Profiling
	// _ "net/http/pprof"

	// *** Included Authorizer plugins
	_ "github.com/lnxjedi/gopherbot/goplugins/groups"

	// *** Included Go plugins, of varying quality; see also: plugin/go-*
	_ "github.com/lnxjedi/gopherbot/goplugins/duo"
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"

	// *** Compiled-in Go Job(s); see also: jobs/go-*
	_ "github.com/lnxjedi/gopherbot/gojobs/go-bootstrap"

	// *** Included Go tasks external to the main engine
	_ "github.com/lnxjedi/gopherbot/gotasks/fail-report"
	_ "github.com/lnxjedi/gopherbot/gotasks/git-command"
	_ "github.com/lnxjedi/gopherbot/gotasks/ssh-agent"
	_ "github.com/lnxjedi/gopherbot/gotasks/ssh-git-helper"
	_ "github.com/lnxjedi/gopherbot/gotasks/update-report"

	// *** Default Slack connector
	_ "github.com/lnxjedi/gopherbot/connectors/slack"

	// *** Default file history
	_ "github.com/lnxjedi/gopherbot/history/file"

	// *** A couple of fantastic brains
	_ "github.com/lnxjedi/gopherbot/brains/dynamodb"
	_ "github.com/lnxjedi/gopherbot/brains/cloudflarekv"
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
