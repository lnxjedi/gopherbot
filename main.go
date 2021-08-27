package main

import "github.com/lnxjedi/gopherbot/bot"

// Version supplied during linking
var Version = "(not set)"

// Commit supplied during linking
var Commit = "(not set)"

func main() {
	versionInfo := bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	}
	bot.Start(versionInfo)
}
