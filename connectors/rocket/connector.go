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
	sync.Mutex
	joinChannels map[string]struct{}
	subChannels  map[string]struct{}
}

var incoming chan models.Message = make(chan models.Message, 100)

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
	rc.Log(bot.Debug, "DEBUG: Raw incoming msg: %v", *msg)
	rc.Log(bot.Debug, "DEBUG: Raw incoming user: %v", *msg.User)
	botMsg := &bot.ConnectorMessage{
		Protocol:      "Rocket",
		UserID:        msg.User.ID,
		ChannelID:     msg.RoomID,
		MessageText:   msg.Msg,
		MessageObject: msg,
		Client:        rc.rt,
	}
	rc.IncomingMessage(botMsg)
}

func (rc *rocketConnector) subscribeChannels() {
	rc.Lock()
	defer rc.Unlock()
	for want := range rc.joinChannels {
		if _, ok := rc.subChannels[want]; !ok {
			rc.subChannels[want] = struct{}{}
			if rid, err := rc.rt.GetChannelId(want); err == nil {
				if err := rc.rt.JoinChannel(rid); err != nil {
					rc.Log(bot.Error, "joining channel %s/%s: %v", want, rid, err)
				} else {
					schan := &models.Channel{ID: rid}
					if err := rc.rt.SubscribeToMessageStream(schan, incoming); err != nil {
						rc.Log(bot.Error, "subscribing to %s/%s: %v", want, rid, err)
					}
				}
			} else {
				rc.Log(bot.Error, "getting channel ID for %s: %v", want, err)
			}
		}
	}
	return
}

func (rc *rocketConnector) sendMessage(ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	return bot.Ok
}
