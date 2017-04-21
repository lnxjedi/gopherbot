// Package slack uses Norberto Lopes' slack library to implement the bot.Connector
// interface.
package slack

import (
	"fmt"
	"log"
	"sync"

	"github.com/nlopes/slack"
	"github.com/uva-its/gopherbot/bot"
)

type config struct {
	SlackToken      string // the 'bot token for connecting to Slack
	MaxMessageSplit int    // the maximum # of ~4000 byte messages to split a large message into
}

var lock sync.Mutex // package var lock
var started bool    // set when connector is started
// Map of bot IDs to unique names; note that for Slack, we have
// to store the botID in the name since bots aren't guaranteed to have
// unique names.
var bots = make(map[string]string)

func init() {
	bot.RegisterConnector("slack", Start)
}

func Start(robot bot.Handler, l *log.Logger) bot.Connector {
	lock.Lock()
	if started {
		lock.Unlock()
		return nil
	}
	started = true
	lock.Unlock()

	var c config

	err := robot.GetProtocolConfig(&c)
	if c.MaxMessageSplit == 0 {
		c.MaxMessageSplit = 1
	}
	if err != nil {
		robot.Log(bot.Fatal, fmt.Errorf("Unable to retrieve protocol configuration: %v", err))
	}

	api := slack.New(c.SlackToken)
	// This spits out a lot of extra stuff, so we only enable it when tracing
	if robot.GetLogLevel() == bot.Trace {
		api.SetDebug(true)
	}
	slack.SetLogger(l)

	sc := &slackConnector{api: api, conn: api.NewRTM(), maxMessageSplit: c.MaxMessageSplit}
	go sc.conn.ManageConnection()

	sc.Handler = robot

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
				for _, b := range ev.Info.Bots {
					if b.Deleted {
						continue
					}
					name := fmt.Sprintf("bot:%s:%s", b.Name, b.ID)
					bots[b.ID] = name
				}
				break Loop

			case *slack.InvalidAuthEvent:
				sc.Log(bot.Fatal, "Invalid credentials")
			}
		}
	}

	sc.updateMaps()
	sc.botFullName = sc.userInfo[sc.botName].RealName
	sc.SetFullName(sc.botFullName)
	sc.Log(bot.Debug, "Set bot full name to", sc.botFullName)

	return bot.Connector(sc)
}

func (sc *slackConnector) Run(stop chan struct{}) {
	sc.Lock()
	// This should never happen, just a bit of defensive coding
	if sc.running {
		sc.Unlock()
		return
	}
	sc.running = true
	sc.Unlock()
loop:
	for {
		select {
		case <-stop:
			sc.Log(bot.Debug, "Received stop in connector")
			break loop
		case msg := <-sc.conn.IncomingEvents:
			sc.Log(bot.Debug, "Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello
			case *slack.ChannelArchiveEvent, *slack.ChannelCreatedEvent, *slack.ChannelDeletedEvent, *slack.ChannelRenameEvent, *slack.TeamJoinEvent, *slack.GroupJoinedEvent, *slack.UserChangeEvent:
				sc.updateMaps()

			case *slack.MessageEvent:
				// Message processing is done concurrently
				go sc.processMessage(ev)

			case *slack.PresenceChangeEvent:
				sc.Log(bot.Debug, fmt.Sprintf("Presence Change: %v", ev))

			case *slack.LatencyReport:
				sc.Log(bot.Debug, fmt.Sprintf("Current latency: %v", ev.Value))

			case *slack.RTMError:
				sc.Log(bot.Debug, fmt.Sprintf("Error: %s\n", ev.Error()))

			default:

				// Ignore other events..
				// robot.Debug(fmt.Sprintf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
