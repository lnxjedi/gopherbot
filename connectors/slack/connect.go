// Package slack uses Norberto Lopes' slack library to implement the bot.Connector
// interface.
package slack

import (
	"log"
	"sync"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/nlopes/slack"
)

type botDefinition struct {
	Name, ID string // e.g. 'mygit', 'BAKDBISDO'
}

type config struct {
	SlackToken      string // the 'bot token for connecting to Slack
	MaxMessageSplit int    // the maximum # of ~4000 byte messages to split a large message into
}

var lock sync.Mutex // package var lock
var started bool    // set when connector is started

func init() {
	bot.RegisterConnector("slack", Initialize)
}

// Initialize starts the connection, sets up and returns the connector object
func Initialize(robot bot.Handler, l *log.Logger) bot.Connector {
	lock.Lock()
	if started {
		lock.Unlock()
		return nil
	}
	started = true
	lock.Unlock()

	var c config
	var tok string

	err := robot.GetProtocolConfig(&c)
	if err != nil {
		robot.Log(bot.Fatal, "unable to retrieve slack protocol configuration: %v", err)
	}

	if c.MaxMessageSplit == 0 {
		c.MaxMessageSplit = 1
	}

	if len(c.SlackToken) == 0 {
		robot.Log(bot.Fatal, "no slack token found in config")
	} else {
		tok = c.SlackToken
	}

	slackOpts := []slack.Option{
		slack.OptionLog(l),
	}
	// This spits out a lot of extra stuff, so we only enable it when tracing
	if robot.GetLogLevel() == bot.Trace {
		slackOpts = append(slackOpts, slack.OptionDebug(true))
	}

	api := slack.New(tok, slackOpts...)

	sc := &slackConnector{
		api:             api,
		conn:            api.NewRTM(),
		maxMessageSplit: c.MaxMessageSplit,
		name:            "slack",
	}
	go sc.conn.ManageConnection()

	sc.Handler = robot

Loop:
	for {
		select {
		case msg := <-sc.conn.IncomingEvents:

			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				sc.Log(bot.Debug, "slack infos: %T %v\n", ev, *ev.Info.User)
				sc.Log(bot.Debug, "slack connection counter: %d", ev.ConnectionCount)
				sc.botName = ev.Info.User.Name
				sc.botID = ev.Info.User.ID
				sc.Log(bot.Info, "slack setting bot internal ID to: %s", sc.botID)
				sc.SetBotID(sc.botID)
				sc.teamID = ev.Info.Team.ID
				sc.Log(bot.Info, "Set team ID to", sc.teamID)
				break Loop

			case *slack.InvalidAuthEvent:
				sc.Log(bot.Fatal, "Invalid credentials")
			}
		}
	}

	sc.updateChannelMaps("")
	sc.updateUserList("")
	sc.botFullName, _ = sc.GetProtocolUserAttribute(sc.botName, "realname")
	go sc.startSendLoop()

	return bot.Connector(sc)
}

func (sc *slackConnector) Run(stop <-chan struct{}) {
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
			sc.Log(bot.Trace, "Event Received (msg, data, type): %v; %v; %T", msg, msg.Data, msg.Data)
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello
			case *slack.ChannelArchiveEvent, *slack.ChannelUnarchiveEvent,
				*slack.ChannelCreatedEvent, *slack.ChannelDeletedEvent,
				*slack.ChannelRenameEvent, *slack.GroupArchiveEvent,
				*slack.GroupUnarchiveEvent, *slack.GroupCreatedEvent,
				*slack.GroupRenameEvent, *slack.IMCloseEvent,
				*slack.IMCreatedEvent, *slack.IMOpenEvent:
				sc.updateChannelMaps("")

			case *slack.MessageEvent:
				// Message processing is done concurrently
				go sc.processMessage(ev)

			case *slack.PresenceChangeEvent:
				sc.Log(bot.Debug, "Presence Change: %v", ev)

			case *slack.LatencyReport:
				sc.Log(bot.Debug, "Current latency: %v", ev.Value)

			case *slack.RTMError:
				sc.Log(bot.Debug, "Error: %s\n", ev.Error())

			default:

				// Ignore other events..
				// robot.Debug(fmt.Sprintf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
