package main

import "github.com/lnxjedi/gopherbot/v2/bot"

// Version supplied during linking
var Version = "(no version set)"

// Commit supplied during linking
var Commit = "(not set)"

func main() {
	versionInfo := bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	}
	bot.Start(versionInfo)
}
