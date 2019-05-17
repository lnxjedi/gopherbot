// Package rocket implements a connector for Rocket.Chat
package rocket

import (
	"sync"

	"github.com/lnxjedi/gopherbot/bot"
	models "github.com/lnxjedi/gopherbot/connectors/rocket/models"
	api "github.com/lnxjedi/gopherbot/connectors/rocket/realtime"
)

type config struct {
	Server   string // Rocket.Chat server to connect to
	Email    string // Rocket.Chat user email
	Password string // the initial userid
}

type rocketConnector struct {
	rt      *api.Client
	user    *models.User
	running bool
	bot.Handler
	sync.RWMutex
	wantChannels map[string]struct{} // channels we want to sub to
	channelNames map[string]string   // map from roomID to channel name
	subChannels  map[string]struct{} // channels we've sub'ed
	dmChannels   map[string]struct{} // direct messages
}

var incoming chan models.Message

func (rc *rocketConnector) Run(stop <-chan struct{}) {
	rc.Lock()
	// This should never happen, just a bit of defensive coding
	if rc.running {
		rc.Unlock()
		return
	}
	rc.running = true
	rc.Unlock()
	rc.subscribeChannels()
loop:
	for {
		select {
		case pmsg := <-incoming:
			rc.processMessage(&pmsg)

		case <-stop:
			rc.Log(bot.Debug, "Received stop in connector")
			break loop
		}
	}
}

// processMessage creates a bot.ConnectorMessage and calls
// bot.IncomingMessage
func (rc *rocketConnector) processMessage(msg *models.Message) {
	rc.Log(bot.Debug, "DEBUG: Raw incoming msg: %+v", *msg)
	rc.Log(bot.Debug, "DEBUG: Raw incoming user: %+v", *msg.User)
	rc.RLock()
	chName := rc.channelNames[msg.RoomID]
	botMsg := &bot.ConnectorMessage{
		Protocol:      "Rocket",
		UserID:        msg.User.ID,
		UserName:      msg.User.UserName,
		ChannelID:     msg.RoomID,
		ChannelName:   chName,
		MessageText:   msg.Msg,
		MessageObject: msg,
		Client:        rc.rt,
	}
	rc.IncomingMessage(botMsg)
}

func (rc *rocketConnector) subscribeChannels() {
	inChannels, err := rc.rt.GetChannelsIn()
	if err != nil {
		rc.Log(bot.Error, "rocket getting channels in: %v", err)
	}
	rc.Lock()
	defer rc.Unlock()
	if len(inChannels) > 0 {
		for _, ich := range inChannels {
			// we want to sub to direct and private; regular
			// channels should be listed in JoinChannels, DefaultChannels,
			// or DefaultJobChannel.
			if ich.Type == "d" || ich.Type == "p" {
				if _, ok := rc.wantChannels[ich.ID]; !ok {
					rc.wantChannels[ich.ID] = struct{}{}
					if len(ich.Name) > 0 {
						rc.channelNames[ich.ID] = ich.Name
					}
				}
			}
		}
	}
	for want := range rc.wantChannels {
		if _, ok := rc.subChannels[want]; !ok {
			chName, ok := rc.channelNames[want]
			if !ok {
				chName = "(private/unknown)"
			}
			rc.Log(bot.Debug, "subscribing to channel %s/%s", chName, want)
			rc.subChannels[want] = struct{}{}
			if err := rc.rt.JoinChannel(want); err != nil {
				rc.Log(bot.Error, "joining channel %s/%s: %v", chName, want, err)
			} else {
				if err := rc.rt.SubscribeRoomUpdates(want); err != nil {
					rc.Log(bot.Error, "subscribing to %s/%s: %v", chName, want, err)
				}
			}
		}
	}
	return
}

func (rc *rocketConnector) sendMessage(ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	return bot.Ok
}
