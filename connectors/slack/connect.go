// Package slack uses Norberto Lopes' slack library to implement the bot.Connector
// interface.
package slack

import (
	"context"
	"log"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type config struct {
	SlackToken         string // the 'bot token for connecting to Slack using RTM
	AppToken, BotToken string // tokens used for connecting to Slack using the new SocketMode
	MaxMessageSplit    int    // the maximum # of ~4000 byte messages to split a large message into
	DisableReflection  bool   // turn off automatic reflection of hidden (slash "/") commands
	Debug              bool   // Explicitly turn on Slack protocol debug output
}

// Initialize validates config, sets up and returns the connector object.
func Initialize(r robot.Handler, l *log.Logger) robot.Connector {
	var c config
	var tok string
	socketMode := false
	enableDebug := false

	slackOpts := []slack.Option{
		slack.OptionLog(l),
	}

	err := r.GetProtocolConfig(&c)
	if err != nil {
		r.Log(robot.Fatal, "Unable to retrieve slack protocol configuration: %v", err)
	}
	// This spits out a lot of extra stuff, so we only enable it when tracing or
	// explicitly configured.
	if c.Debug || r.GetLogLevel() == robot.Trace {
		slackOpts = append(slackOpts, slack.OptionDebug(true))
		enableDebug = true
	}

	if c.MaxMessageSplit == 0 {
		c.MaxMessageSplit = 1
	}

	if len(c.BotToken) > 0 && len(c.AppToken) > 0 {
		if !strings.HasPrefix(c.BotToken, "xoxb-") {
			r.Log(robot.Fatal, "BotToken must have the prefix \"xoxb-\".")
		}
		if !strings.HasPrefix(c.AppToken, "xapp-") {
			r.Log(robot.Fatal, "AppToken must have the prefix \"xapp-\".")
		}
		tok = c.BotToken
		socketMode = true
		slackOpts = append(slackOpts, slack.OptionAppLevelToken(c.AppToken))
	} else {
		if len(c.SlackToken) == 0 {
			r.Log(robot.Fatal, "No slack token or bot/app tokens found in config")
		} else {
			if !strings.HasPrefix(c.SlackToken, "xoxb-") {
				r.Log(robot.Fatal, "BotToken must have the prefix \"xoxb-\".")
			}
			r.Log(robot.Warn, "Using deprecated legacy RTM mode for connection")
			tok = c.SlackToken
		}
	}

	api := slack.New(tok, slackOpts...)

	sc := &slackConnector{
		api:             api,
		maxMessageSplit: c.MaxMessageSplit,
		reflectHidden:   !c.DisableReflection,
		name:            "slack",
		socketMode:      socketMode,
		sendQueue:       nil,
	}
	if socketMode {
		sockOpts := []socketmode.Option{
			socketmode.OptionLog(l),
			socketmode.OptionDebug(enableDebug),
		}
		sc.sock = socketmode.New(api, sockOpts...)
	} else {
		sc.conn = api.NewRTM()
	}

	sc.Handler = r

	info, err := api.AuthTest()
	if err != nil {
		r.Log(robot.Fatal, "Error getting auth info: %v", err)
	}
	r.Log(robot.Debug, "Retrieved auth info:\n%+v", info)
	sc.botUserID = info.UserID
	r.Log(robot.Info, "Slack setting bot internal ID to: %s", sc.botUserID)
	r.SetBotID(sc.botUserID)
	sc.botID = info.BotID
	sc.botName = info.User
	sc.teamID = info.TeamID
	r.Log(robot.Info, "Set team ID to %s", sc.teamID)
	botInfo, err := api.GetBotInfo(slack.GetBotInfoParameters{
		Bot:    sc.botID,
		TeamID: sc.teamID,
	})
	if err != nil {
		r.Log(robot.Fatal, "Error getting bot info: %v", err)
	}
	sc.botFullName = botInfo.Name

	sc.updateChannelMaps("")
	// This should trigger from the engine calling SetUserMap
	// sc.updateUserList("")

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
	sc.sendQueue = make(chan *sendMessage, sendQueueSize)
	sc.lastMsgTime = nil
	sc.Unlock()

	sendStop := make(chan struct{})
	go sc.startSendLoop(sendStop)

	defer func() {
		close(sendStop)
		sc.Lock()
		sc.running = false
		sc.sendQueue = nil
		sc.Unlock()
	}()

	if sc.socketMode {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			if err := sc.sock.RunContext(ctx); err != nil && ctx.Err() == nil {
				sc.Log(robot.Error, "Slack socket mode runtime failed: %v", err)
			}
		}()
	SOCKRunLoop:
		for {
			select {
			case <-stop:
				sc.Log(robot.Debug, "Received stop in connector")
				cancel()
				break SOCKRunLoop
			case evt, ok := <-sc.sock.Events:
				if !ok {
					sc.Log(robot.Error, "Slack socket mode event channel closed")
					break SOCKRunLoop
				}
				switch evt.Type {
				case socketmode.EventTypeConnected:
					connectEvent, ok := evt.Data.(*socketmode.ConnectedEvent)
					if !ok {
						sc.Log(robot.Warn, "Ignoring %+v", evt)
					} else {
						sc.Log(robot.Debug, "Socket mode connected to '%s', count: %d",
							connectEvent.Info.URL,
							connectEvent.ConnectionCount)
					}
				case socketmode.EventTypeHello:
					if evt.Request != nil {
						sc.Log(robot.Debug, "Received hello event for app_id '%s', slack host '%s', build number: %d",
							evt.Request.ConnectionInfo.AppID,
							evt.Request.DebugInfo.Host,
							evt.Request.DebugInfo.BuildNumber)
						sc.appID = evt.Request.ConnectionInfo.AppID
					}
				case socketmode.EventTypeInvalidAuth:
					sc.Log(robot.Error, "Invalid Slack credentials")
					break SOCKRunLoop
				case socketmode.EventTypeEventsAPI:
					eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
					if !ok {
						sc.Log(robot.Warn, "Ignored %+v", evt)
						continue
					}
					sc.Log(robot.Trace, "Event received: %+v", eventsAPIEvent)
					if evt.Request != nil {
						sc.sock.Ack(*evt.Request)
					}

					switch eventsAPIEvent.Type {
					case slackevents.CallbackEvent:
						innerEvent := eventsAPIEvent.InnerEvent
						switch innerEvent.Type {
						case "channel_archive", "channel_unarchive",
							"channel_created", "channel_deleted",
							"channel_rename", "channel_id_changed",
							"group_archive", "group_deleted",
							"group_open", "group_rename",
							"im_created", "im_open",
							"im_close":
							sc.updateChannelMaps("")
						case "message":
							mevt, ok := innerEvent.Data.(*slackevents.MessageEvent)
							if !ok {
								sc.Log(robot.Warn, "Ignoring message event with unexpected type: %T", innerEvent.Data)
								continue
							}
							go sc.processMessageSocketMode(mevt)
						default:
							sc.Log(robot.Debug, "Ignored CallbackEvent type: %s", innerEvent.Type)
						}
					default:
						sc.Log(robot.Debug, "Unhandled Events API event received, type: %s", eventsAPIEvent.Type)
					}
				case socketmode.EventTypeSlashCommand:
					cmd, ok := evt.Data.(slack.SlashCommand)
					if !ok {
						sc.Log(robot.Warn, "Ignored %+v", evt)
						continue
					}
					if evt.Request != nil {
						sc.sock.Ack(*evt.Request)
					}
					go sc.processSlashCmdSocketMode(&cmd)
				case socketmode.EventTypeInteractive:
					if evt.Request != nil {
						sc.sock.Ack(*evt.Request)
					}
				default:
					sc.Log(robot.Debug, "Ignoring event type: %s", evt.Type)
				}
			}
		}
	} else {
		sc.Lock()
		sc.conn = sc.api.NewRTM()
		sc.Unlock()
		go sc.conn.ManageConnection()
		defer func() {
			if sc.conn != nil {
				if err := sc.conn.Disconnect(); err != nil {
					sc.Log(robot.Warn, "Slack RTM disconnect failed: %v", err)
				}
			}
		}()
	RTMRunLoop:
		for {
			select {
			case <-stop:
				sc.Log(robot.Debug, "Received stop in connector")
				break RTMRunLoop
			case msg, ok := <-sc.conn.IncomingEvents:
				if !ok {
					sc.Log(robot.Error, "Slack RTM event channel closed")
					break RTMRunLoop
				}
				sc.Log(robot.Trace, "Event Received (msg, data, type): %v; %v; %T", msg, msg.Data, msg.Data)
				switch ev := msg.Data.(type) {
				case *slack.ChannelArchiveEvent, *slack.ChannelUnarchiveEvent,
					*slack.ChannelCreatedEvent, *slack.ChannelDeletedEvent,
					*slack.ChannelRenameEvent, *slack.GroupArchiveEvent,
					*slack.GroupUnarchiveEvent, *slack.GroupCreatedEvent,
					*slack.GroupRenameEvent, *slack.IMCloseEvent,
					*slack.IMCreatedEvent, *slack.IMOpenEvent:
					sc.updateChannelMaps("")
				case *slack.ConnectedEvent:
					sc.Log(robot.Debug, "Slack connected, count: %d", ev.ConnectionCount)
				case *slack.InvalidAuthEvent:
					sc.Log(robot.Error, "Invalid Slack credentials")
					break RTMRunLoop

				case *slack.MessageEvent:
					// Message processing is done concurrently
					go sc.processMessageRTM(ev)

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
}
