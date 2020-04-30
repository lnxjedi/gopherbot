package main

import "github.com/lnxjedi/gopherbot/bot"

// Version of gopherbot
// NOTE NOTE NOTE: update docker builds.
var Version = "v2.0.0-beta4-snapshot"

// Commit supplied during linking
var Commit = "(not set)"

func main() {
	versionInfo := bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	}
	bot.Start(versionInfo)
}
