// Package slack uses Norberto Lopes' slack library to implement the bot.Connector
// interface.
package slack

import (
	"log"
	"strings"
	"sync"

	"github.com/lnxjedi/robot"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type botDefinition struct {
	Name, ID string // e.g. 'mygit', 'BAKDBISDO'
}

type config struct {
	SlackToken         string // the 'bot token for connecting to Slack using RTM
	AppToken, BotToken string // tokens used for connecting to Slack using the new SocketMode
	MaxMessageSplit    int    // the maximum # of ~4000 byte messages to split a large message into
}

var lock sync.Mutex        // package var lock
var started bool           // set when connector is started
var socketmodeEnabled bool // set when using socketmode to connect, duh
var slackDebug bool        // set to enable debugging output in slack lib

// Initialize starts the connection, sets up and returns the connector object
func Initialize(r robot.Handler, l *log.Logger) robot.Connector {
	lock.Lock()
	if started {
		lock.Unlock()
		return nil
	}
	started = true
	lock.Unlock()

	var c config
	var tok string

	slackOpts := []slack.Option{
		slack.OptionLog(l),
	}
	// This spits out a lot of extra stuff, so we only enable it when tracing
	if r.GetLogLevel() == robot.Trace {
		slackOpts = append(slackOpts, slack.OptionDebug(true))
		slackDebug = true
	}

	err := r.GetProtocolConfig(&c)
	if err != nil {
		r.Log(robot.Fatal, "unable to retrieve slack protocol configuration: %v", err)
	}

	if c.MaxMessageSplit == 0 {
		c.MaxMessageSplit = 1
	}

	if len(c.SlackToken) == 0 {
		if len(c.BotToken) > 0 && len(c.AppToken) > 0 {
			if !strings.HasPrefix(c.BotToken, "xoxb-") {
				r.Log(robot.Fatal, "BotToken must have the prefix \"xoxb-\".")
			}
			if !strings.HasPrefix(c.AppToken, "xapp-") {
				r.Log(robot.Fatal, "AppToken must have the prefix \"xapp-\".")
			}
			tok = c.BotToken
			socketmodeEnabled = true
			slackOpts = append(slackOpts, slack.OptionAppLevelToken(c.AppToken))
		} else {
			r.Log(robot.Fatal, "no slack token or bot/app tokens found in config")
		}
	} else {
		tok = c.SlackToken
	}

	api := slack.New(tok, slackOpts...)

	var sc *slackConnector

	if socketmodeEnabled {
		sockOpts := []socketmode.Option{
			socketmode.OptionLog(l),
			socketmode.OptionDebug(slackDebug),
		}
		sc = &slackConnector{
			api:             api,
			sock:            socketmode.New(api, sockOpts...),
			maxMessageSplit: c.MaxMessageSplit,
			name:            "slack",
		}
		go sc.sock.Run()
	} else {
		sc = &slackConnector{
			api:             api,
			conn:            api.NewRTM(),
			maxMessageSplit: c.MaxMessageSplit,
			name:            "slack",
		}
		go sc.conn.ManageConnection()
	}

	sc.Handler = r

	if socketmodeEnabled {
	SOCKLoop:
		for evt := range sc.sock.Events {
			switch evt.Type {
			case socketmode.EventTypeConnected:
				connectEvent, ok := evt.Data.(*socketmode.ConnectedEvent)
				if !ok {
					r.Log(robot.Warn, "Ignoring %+v", evt)
				} else {
					r.Log(robot.Debug, "Socket mode connected to '%s', count: %d",
						connectEvent.Info.URL,
						connectEvent.ConnectionCount)
				}
			case socketmode.EventTypeDisconnect:
				break SOCKLoop // DEBUG
			case socketmode.EventTypeHello:
				r.Log(robot.Debug, "Received hello event for app_id '%s', slack host '%s', build number: %d",
					evt.Request.ConnectionInfo.AppID,
					evt.Request.DebugInfo.Host,
					evt.Request.DebugInfo.BuildNumber)
				sc.appID = evt.Request.ConnectionInfo.AppID
				break SOCKLoop
			case socketmode.EventTypeInvalidAuth:
				r.Log(robot.Fatal, "Invalid credentials")
			default:
				if evt.Request == nil {
					r.Log(robot.Debug, "Unhandled event type '%s' (nil request)", evt.Type)
				} else {
					r.Log(robot.Debug, "Unhandled event type '%s':\n%v", evt.Type, evt.Request)
				}
			}
		}
	} else {
	RTMLoop:
		for msg := range sc.conn.IncomingEvents {
			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				r.Log(robot.Debug, "slack infos: %T %v\n", ev, *ev.Info.User)
				r.Log(robot.Debug, "slack connection counter: %d", ev.ConnectionCount)
				sc.botName = ev.Info.User.Name
				sc.botUserID = ev.Info.User.ID
				r.Log(robot.Info, "slack setting bot internal ID to: %s", sc.botID)
				r.SetBotID(sc.botID)
				sc.teamID = ev.Info.Team.ID
				r.Log(robot.Info, "Set team ID to %s", sc.teamID)
				break RTMLoop

			case *slack.InvalidAuthEvent:
				r.Log(robot.Fatal, "Invalid credentials")
			}
		}
	}

	info, err := api.AuthTest()
	if err != nil {
		r.Log(robot.Fatal, "Error getting auth info: %v", err)
	}
	r.Log(robot.Debug, "retrieved auth info:\n%+v", info)
	sc.botUserID = info.UserID
	r.Log(robot.Info, "slack setting bot internal ID to: %s", sc.botUserID)
	r.SetBotID(sc.botUserID)
	sc.botID = info.BotID
	botInfo, err := api.GetBotInfo(sc.botID)
	if err != nil {
		r.Log(robot.Fatal, "Error getting bot info: %v", err)
	}
	r.Log(robot.Debug, "retrieved bot info:\n%+v", botInfo)
	// 	sc.teamID = ev.Info.Team.ID
	// 	r.Log(robot.Info, "Set team ID to %s", sc.teamID)

	r.Log(robot.Fatal, "DEBUG testing socketmode support")

	sc.updateChannelMaps("")
	sc.updateUserList("")
	sc.botFullName, _ = sc.GetProtocolUserAttribute(sc.botName, "realname")
	go sc.startSendLoop()

	return robot.Connector(sc)
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
			sc.Log(robot.Debug, "Received stop in connector")
			break loop
		case msg := <-sc.conn.IncomingEvents:
			sc.Log(robot.Trace, "Event Received (msg, data, type): %v; %v; %T", msg, msg.Data, msg.Data)
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
				sc.Log(robot.Debug, "Presence Change: %v", ev)

			case *slack.LatencyReport:
				sc.Log(robot.Debug, "Current latency: %v", ev.Value)

			case *slack.RTMError:
				sc.Log(robot.Debug, "Error: %s\n", ev.Error())

			default:

				// Ignore other events..
				// robot.Debug(fmt.Sprintf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
