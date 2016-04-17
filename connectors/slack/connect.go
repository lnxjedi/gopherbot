// package slackConnector connects gobot to Slack and implements
// the gobot Connector interface
package slack

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nlopes/slack"
	"github.com/parsley42/gobot/bot"
)

type config struct {
	SlackToken string // the 'bot token for connecting to Slack
}

func StartConnector(gobot *bot.Bot) {
	var c config

	configJSON := gobot.GetProtocolConfig()
	json.Unmarshal(configJSON, &c)

	api := slack.New(c.SlackToken)
	if gobot.GetLogLevel() <= bot.Debug {
		api.SetDebug(true)
	}

	sc := &slackConnector{api: api, conn: api.NewRTM()}
	go sc.conn.ManageConnection()

Loop:
	for {
		select {
		case msg := <-sc.conn.IncomingEvents:

			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				gobot.Log(bot.Debug, fmt.Sprintf("Infos: %T %v\n", ev, *ev.Info.User))
				gobot.Log(bot.Debug, "Connection counter:", ev.ConnectionCount)
				sc.botName = ev.Info.User.Name
				gobot.SetName(sc.botName)
				gobot.Log(bot.Debug, "Set bot name to", sc.botName)
				sc.botID = ev.Info.User.ID
				gobot.Log(bot.Trace, "Set bot ID to", sc.botID)
				break Loop

			case *slack.InvalidAuthEvent:
				log.Fatalln("Invalid credentials")
			}
		}
	}

	sc.Handler = gobot
	sc.updateMaps()
	// We're connected, set the bot's connector to a struct
	gobot.Init(sc)

	for {
		select {
		case msg := <-sc.conn.IncomingEvents:
			gobot.Log(bot.Debug, "Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello
			case *slack.ChannelArchiveEvent, *slack.ChannelCreatedEvent, *slack.ChannelDeletedEvent, *slack.ChannelRenameEvent, *slack.TeamJoinEvent:
				sc.updateMaps()

			case *slack.MessageEvent:
				sc.processMessage(ev)

			case *slack.PresenceChangeEvent:
				gobot.Log(bot.Debug, fmt.Sprintf("Presence Change: %v\n", ev))

			case *slack.LatencyReport:
				gobot.Log(bot.Debug, fmt.Sprintf("Current latency: %v\n", ev.Value))

			case *slack.RTMError:
				gobot.Log(bot.Debug, fmt.Sprintf("Error: %s\n", ev.Error()))

			default:

				// Ignore other events..
				// gobot.Debug(fmt.Sprintf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
