// package slackConnector connects gobot to Slack and implements
// the gobot Connector interface
package slack

import (
	"fmt"
	"log"

	"github.com/nlopes/slack"
	"github.com/parsley42/gobot/bot"
)

type slackConnector struct {
	api  *slack.Client
	conn *slack.RTM
}

func (s *slackConnector) SendChannelMessage(c string, m string) {
	s.conn.SendMessage(s.conn.NewOutgoingMessage(m, c))
}

func StartConnector(bot *gobot.Bot, token string) {
	api := slack.New(token)
	if bot.GetDebug() {
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
				bot.Debug("Infos:", ev.Info)
				bot.Debug("Connection counter:", ev.ConnectionCount)
				// bot.SetName(name)
				// Replace #general with your Channel ID
				//	sc.conn.SendMessage(sc.conn.NewOutgoingMessage("Hello world", "C0RK4DG68"))
				break Loop

			case *slack.InvalidAuthEvent:
				log.Fatalln("Invalid credentials")
			}
		}
	}

	// We're connected, set the bot's connector to a struct

	bot.Init(sc)

	for {
		select {
		case msg := <-sc.conn.IncomingEvents:
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
