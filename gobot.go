package main

import (
	"log"
	"os"
	"unicode/utf8"

	//	_ "github.com/parsley42/gobot-instance/plugin"
	"github.com/parsley42/gobot/bot"
	"github.com/parsley42/gobot/connectors/slack"
)

func main() {
	//	bot := gobot.New(string(alias), os.Getenv("GOBOT_HTTP_PORT"), debug)
	bot := gobot.New()

	alias := ';'
	if len(os.Getenv("GOBOT_ALIAS")) > 0 {
		alias, _ = utf8.DecodeRuneInString(os.Getenv("GOBOT_ALIAS"))
	}
	bot.SetAlias(alias)

	debug := false
	if os.Getenv("GOBOT_DEBUG") == "true" {
		debug = true
	}
	bot.SetDebug(debug)

	bot.SetPort(os.Getenv("GOBOT_HTTP_PORT"))

	switch os.Getenv("GOBOT_CONNECTOR") {
	case "slack":
		if len(os.Getenv("GOBOT_SLACK_TOKEN")) == 0 {
			log.Fatal("\"slack\" GOBOT_CONNECTOR requires GOBOT_SLACK_TOKEN")
		}
		slack.StartConnector(bot, os.Getenv("GOBOT_SLACK_TOKEN"))
	default:
		log.Fatalln("Unsupported/unknown GOBOT_CONNECTOR: " + os.Getenv("GOBOT_CONNECTOR"))
	}
}
