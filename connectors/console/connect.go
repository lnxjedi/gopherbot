// Package term implements a terminal console connector for plugin development
// bot testing; eventually a test framework will be built around it.
package term

import (
	"fmt"
	"log"
	"sync"

	"github.com/lnxjedi/gopherbot/bot"
)

type config struct {
	StartChannel string // the initial channel
	StartUser    string // the initial userid
}

// termConnector holds all the relevant data about a connection
type termConnector struct {
	currentChannel string // The current channel for the user
	curentUser     string //
	running        bool   // set on call to Run
	botName        string // human-readable name of bot
	botFullName    string // human-readble full name of the bot
	botID          string // slack internal bot ID
	bot.Handler           // bot API for connectors
	sync.RWMutex          // shared mutex for locking connector data structures
}

var lock sync.Mutex // package var lock
var started bool    // set when connector is started

func init() {
	bot.RegisterConnector("term", Start)
}

// Start starts the connector
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
	if err != nil {
		robot.Log(bot.Fatal, fmt.Errorf("Unable to retrieve protocol configuration: %v", err))
	}

	api := slack.New(c.SlackToken)
	// This spits out a lot of extra stuff, so we only enable it when tracing
	if robot.GetLogLevel() == bot.Trace {
		api.SetDebug(true)
	}
	slack.SetLogger(l)

	sc := &termConnector{api: api, conn: api.NewRTM(), maxMessageSplit: c.MaxMessageSplit}
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
	go sc.startSendLoop()

	return bot.Connector(sc)
}

func (sc *termConnector) Run(stop chan struct{}) {
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
			sc.Log(bot.Trace, fmt.Sprintf("Event Received (msg, data, type): %v; %v; %T", msg, msg.Data, msg.Data))
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
