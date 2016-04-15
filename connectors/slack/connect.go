// package slackConnector connects gobot to Slack and implements
// the gobot Connector interface
package slack

import (
	"fmt"
	"log"

	"github.com/nlopes/slack"
	"github.com/parsley42/gobot/bot"
)

func StartConnector(gobot *bot.Bot, token string, l bot.LogLevel) {
	api := slack.New(token)
	if gobot.GetDebug() {
		api.SetDebug(true)
	}

	sc := &slackConnector{api: api, conn: api.NewRTM(), level: l}
	go sc.conn.ManageConnection()

Loop:
	for {
		select {
		case msg := <-sc.conn.IncomingEvents:

			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				gobot.Debug(fmt.Sprintf("Infos: %T %v\n", ev, *ev.Info.User))
				gobot.Debug("Connection counter:", ev.ConnectionCount)
				gobot.SetName(ev.Info.User.Name)
				sc.botName = ev.Info.User.Name
				sc.botID = ev.Info.User.ID
				sc.log(bot.Trace, "Set bot ID to", sc.botID)
				break Loop

			case *slack.InvalidAuthEvent:
				log.Fatalln("Invalid credentials")
			}
		}
	}

	sc.updateMaps()
	// We're connected, set the bot's connector to a struct
	gobot.Init(sc)
	sc.Handler = gobot

	for {
		select {
		case msg := <-sc.conn.IncomingEvents:
			gobot.Debug("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello
			case *slack.ChannelArchiveEvent, *slack.ChannelCreatedEvent, *slack.ChannelDeletedEvent, *slack.ChannelRenameEvent, *slack.TeamJoinEvent:
				sc.updateMaps()

			case *slack.MessageEvent:
				sc.processMessage(ev)

			case *slack.PresenceChangeEvent:
				gobot.Debug(fmt.Sprintf("Presence Change: %v\n", ev))

			case *slack.LatencyReport:
				gobot.Debug(fmt.Sprintf("Current latency: %v\n", ev.Value))

			case *slack.RTMError:
				gobot.Debug(fmt.Sprintf("Error: %s\n", ev.Error()))

			default:

				// Ignore other events..
				// gobot.Debug(fmt.Sprintf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
