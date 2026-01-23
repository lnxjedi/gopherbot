//go:build test
// +build test

package main

import (
	"github.com/lnxjedi/gopherbot/bot"
	_ "github.com/lnxjedi/gopherbot/history/file"
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"
	_ "github.com/lnxjedi/gopherbot/connectors/test"
)

// Version and Commit are set by the build process
var Version = "v2.16.0-snapshot"
var Commit = ""

func main() {
	bot.Start(bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	})
}
