package main

import (
	"log"
	"os"
	"strings"
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

	bot.SetInitChannels(strings.Split(os.Getenv("GOBOT_JOIN_CHANNELS"), " "))

	var loglevel gobot.LogLevel
	switch os.Getenv("GOBOT_LOGLEVEL") {
	case "trace":
		loglevel = gobot.Trace
	case "debug":
		loglevel = gobot.Debug
	case "info":
		loglevel = gobot.Info
	case "warn":
		loglevel = gobot.Warn
	default:
		loglevel = gobot.Error
	}
	log.Println("Setting log level to ", loglevel)

	switch os.Getenv("GOBOT_CONNECTOR") {
	case "slack":
		if len(os.Getenv("GOBOT_SLACK_TOKEN")) == 0 {
			log.Fatal("\"slack\" GOBOT_CONNECTOR requires GOBOT_SLACK_TOKEN")
		}
		slack.StartConnector(bot, os.Getenv("GOBOT_SLACK_TOKEN"), loglevel)
	default:
		log.Fatalln("Unsupported/unknown GOBOT_CONNECTOR: " + os.Getenv("GOBOT_CONNECTOR"))
	}
}
