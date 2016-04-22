// Package slack uses a slack library to implement the methods
// required by gobot-chatops and it's plugins.
package slack

import (
	"fmt"
	"log"
	"sync"

	"github.com/nlopes/slack"
	"github.com/parsley42/gopherbot/bot"
)

type Config struct {
	SlackToken string // the 'bot token for connecting to Slack
}

var lock sync.Mutex // package var lock
var started bool    // set when connector is started

func Start(gobot bot.Handler) bot.Connector {
	lock.Lock()
	if started {
		lock.Unlock()
		return nil
	}
	started = true
	lock.Unlock()

	var c Config

	err := gobot.GetProtocolConfig(&c)
	if err != nil {
		log.Fatal(fmt.Errorf("Unable to retrieve protocol configuration: %v", err))
	}

	api := slack.New(c.SlackToken)
	if gobot.GetLogLevel() <= bot.Debug {
		api.SetDebug(true)
	}

	sc := &slackConnector{api: api, conn: api.NewRTM()}
	go sc.conn.ManageConnection()

	sc.Handler = gobot

Loop:
	for {
		select {
		case msg := <-sc.conn.IncomingEvents:

			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				sc.Log(bot.Debug, fmt.Sprintf("Infos: %T %v\n", ev, *ev.Info.User))
				sc.Log(bot.Debug, "Connection counter:", ev.ConnectionCount)
				sc.botName = ev.Info.User.Name
				sc.SetName(sc.botName)
				sc.Log(bot.Debug, "Set bot name to", sc.botName)
				sc.botID = ev.Info.User.ID
				sc.Log(bot.Trace, "Set bot ID to", sc.botID)
				break Loop

			case *slack.InvalidAuthEvent:
				log.Fatalln("Invalid credentials")
			}
		}
	}

	sc.updateMaps()

	return bot.Connector(sc)
}

func (sc *slackConnector) Run() {
	sc.Lock()
	// This should never happen, just a bit of defensive coding
	if sc.running {
		sc.Unlock()
		return
	}
	sc.running = true
	sc.Unlock()
	for {
		select {
		case msg := <-sc.conn.IncomingEvents:
			sc.Log(bot.Debug, "Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello
			case *slack.ChannelArchiveEvent, *slack.ChannelCreatedEvent, *slack.ChannelDeletedEvent, *slack.ChannelRenameEvent, *slack.TeamJoinEvent:
				sc.updateMaps()

			case *slack.MessageEvent:
				// Message processing is done concurrently
				go sc.processMessage(ev)

			case *slack.PresenceChangeEvent:
				sc.Log(bot.Debug, fmt.Sprintf("Presence Change: %v\n", ev))

			case *slack.LatencyReport:
				sc.Log(bot.Debug, fmt.Sprintf("Current latency: %v\n", ev.Value))

			case *slack.RTMError:
				sc.Log(bot.Debug, fmt.Sprintf("Error: %s\n", ev.Error()))

			default:

				// Ignore other events..
				// gobot.Debug(fmt.Sprintf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
