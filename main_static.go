// +build !modular test

package main

import (
	"github.com/lnxjedi/gopherbot/bot"
	// *** Included Authorizer plugins
	_ "github.com/lnxjedi/gopherbot/goplugins/groups"

	// *** Included Go plugins, of varying quality
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"
)

// Version of gopherbot
var Version = "v2.0.0-snapshot"

// Commit supplied during linking
var Commit = "(not set)"

func main() {
	versionInfo := bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	}
	bot.Start(versionInfo)
}
