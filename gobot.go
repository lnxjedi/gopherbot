package main

import (
	"log"
	"os"
	"unicode/utf8"

	"github.com/parsley42/gobot/bot"
	"github.com/parsley42/gobot/slackConnector"
	//	_ "github.com/parsley42/gobot-instance/plugin"
)

func main() {
	alias := ';'

	if len(os.Getenv("GOBOT_ALIAS")) > 0 {
		alias, _ = utf8.DecodeRuneInString(os.Getenv("GOBOT_ALIAS"))
	}

	debug := false
	if os.Getenv("GOBOT_DEBUG") == "true" {
		debug = true
	}

	bot := gobot.New(string(alias), debug)

	switch os.Getenv("GOBOT_CONNECTOR") {
	case "slack":
		if len(os.Getenv("GOBOT_SLACK_TOKEN")) == 0 {
			log.Fatal("\"slack\" GOBOT_CONNECTOR requires GOBOT_SLACK_TOKEN")
		}
		slackConnector.Start(bot, os.Getenv("GOBOT_SLACK_TOKEN"))
	default:
		log.Fatalln("Unsupport/unknown GOBOT_CONNECTOR: " + os.Getenv("GOBOT_CONNECTOR"))
	}
}
