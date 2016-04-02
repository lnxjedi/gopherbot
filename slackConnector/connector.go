package slackConnector

import (
	"fmt"
	"log"

	"github.com/nlopes/slack"
	"github.com/parsley42/gobot/bot"
)

func Start(bot *gobot.Bot, token string) {
	api := slack.New(token)
	if bot.GetDebug() {
		api.SetDebug(true)
	}

	rtm := api.NewRTM()
	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:

			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				bot.Debug("Infos:", ev.Info)
				bot.Debug("Connection counter:", ev.ConnectionCount)
				// Replace #general with your Channel ID
				rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "C0RK4DG68"))
				break Loop

			case *slack.InvalidAuthEvent:
				log.Fatalln("Invalid credentials")
			}
		}
	}

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			bot.Debug("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello

			case *slack.MessageEvent:
				bot.Debug(fmt.Sprintf("Message: %v\n", ev))

			case *slack.PresenceChangeEvent:
				bot.Debug(fmt.Sprintf("Presence Change: %v\n", ev))

			case *slack.LatencyReport:
				bot.Debug(fmt.Sprintf("Current latency: %v\n", ev.Value))

			case *slack.RTMError:
				bot.Debug(fmt.Sprintf("Error: %s\n", ev.Error()))

			default:

				// Ignore other events..
				// bot.Debug(fmt.Sprintf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
