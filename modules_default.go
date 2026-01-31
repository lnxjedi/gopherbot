//go:build !test
// +build !test

package main

// modules_default.go - blank imports for all the modules that should be compiled in.

import (
	// Profiling
	// _ "net/http/pprof"

	// *** Included Authorizer plugins
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/groups"

	// *** Included Go plugins, of varying quality; see also: plugin/go-*
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/duo"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/ping"

	// *** Compiled-in Go Job(s); see also: jobs/go-*
	_ "github.com/lnxjedi/gopherbot/v2/gojobs/go-bootstrap"

	// *** Included Go tasks external to the main engine
	_ "github.com/lnxjedi/gopherbot/v2/gotasks/fail-report"
	_ "github.com/lnxjedi/gopherbot/v2/gotasks/git-command"
	_ "github.com/lnxjedi/gopherbot/v2/gotasks/ssh-agent"
	_ "github.com/lnxjedi/gopherbot/v2/gotasks/ssh-git-helper"
	_ "github.com/lnxjedi/gopherbot/v2/gotasks/update-report"

	// *** Default Slack connector
	_ "github.com/lnxjedi/gopherbot/v2/connectors/slack"
	// *** Default SSH connector
	_ "github.com/lnxjedi/gopherbot/v2/connectors/ssh"

	// *** Default file history
	_ "github.com/lnxjedi/gopherbot/v2/history/file"

	// *** A couple of fantastic brains
	_ "github.com/lnxjedi/gopherbot/v2/brains/cloudflarekv"
	_ "github.com/lnxjedi/gopherbot/v2/brains/dynamodb"
)
