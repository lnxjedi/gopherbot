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
	channelNames   map[string]string   // map from roomID to channel name
	channelIDs     map[string]string   // map from channel name to roomID
	joinedChannels map[string]struct{} // channels we've joined
	dmChannels     map[string]struct{} // direct messages
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
	rc.updateChannels()
	if err := rc.rt.SubscribeRoomUpdates("__my_messages__"); err != nil {
		rc.Log(bot.Error, "failed subscribing to '__my_messages__, won't hear messages: %v", err)
	}
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
	if len(msg.Msg) == 0 {
		return
	}
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

func (rc *rocketConnector) updateChannels() {
	inChannels, err := rc.rt.GetChannelsIn()
	if err != nil {
		rc.Log(bot.Error, "rocket getting channels in: %v", err)
	}
	rc.Lock()
	defer rc.Unlock()
	if len(inChannels) > 0 {
		for _, ich := range inChannels {
			if len(ich.Name) > 0 {
				rc.channelNames[ich.ID] = ich.Name
				rc.channelIDs[ich.Name] = ich.ID
			}
			if ich.Type == "d" {
				rc.dmChannels[ich.ID] = struct{}{}
			}
		}
	}
}

func (rc *rocketConnector) sendMessage(ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	return bot.Ok
}
