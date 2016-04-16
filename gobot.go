package main

import (
	"log"
	"os"
	"strings"
	"unicode/utf8"

	//	_ "github.com/parsley42/gobot-instance/plugin"
	"github.com/parsley42/gobot/bot"
	"github.com/parsley42/gobot/connectors/slack"
	_ "github.com/parsley42/gobot/plugins/meme"
)

func main() {
	//	bot := gobot.New(string(alias), os.Getenv("GOBOT_HTTP_PORT"), debug)
	gobot := bot.Create()

	if len(os.Getenv("GOBOT_ALIAS")) > 0 {
		alias, _ := utf8.DecodeRuneInString(os.Getenv("GOBOT_ALIAS"))
		gobot.SetAlias(alias)
	}

	debug := false
	if os.Getenv("GOBOT_DEBUG") == "true" {
		debug = true
	}
	gobot.SetDebug(debug)

	if len(os.Getenv("GOBOT_LOCAL_PORT")) > 0 {
		gobot.SetPort("127.0.0.1:" + os.Getenv("GOBOT_LOCAL_PORT"))
	}

	gobot.SetInitChannels(strings.Split(os.Getenv("GOBOT_JOIN_CHANNELS"), " "))

	var loglevel bot.LogLevel
	switch os.Getenv("GOBOT_LOGLEVEL") {
	case "trace":
		loglevel = bot.Trace
	case "debug":
		loglevel = bot.Debug
	case "info":
		loglevel = bot.Info
	case "warn":
		loglevel = bot.Warn
	default:
		loglevel = bot.Error
	}
	log.Println("Setting log level to ", loglevel)

	switch os.Getenv("GOBOT_CONNECTOR") {
	case "slack":
		if len(os.Getenv("GOBOT_SLACK_TOKEN")) == 0 {
			log.Fatal("\"slack\" GOBOT_CONNECTOR requires GOBOT_SLACK_TOKEN")
		}
		slack.StartConnector(gobot, os.Getenv("GOBOT_SLACK_TOKEN"), loglevel)
	default:
		log.Fatalln("Unsupported/unknown GOBOT_CONNECTOR: " + os.Getenv("GOBOT_CONNECTOR"))
	}
}
