// Package rocket implements a connector for Rocket.Chat
package rocket

import (
	"github.com/lnxjedi/gopherbot/bot"
	models "github.com/lnxjedi/gopherbot/connectors/rocket/models"
)

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
	if len(msg.Msg) == 0 {
		return
	}
	hearIt := false
	directMsg := false
	mapUser := false
	rc.RLock()
	chName := rc.channelNames[msg.RoomID]
	if _, ok := rc.dmChannels[msg.RoomID]; ok {
		hearIt = true
		directMsg = true
	}
	if _, ok := rc.privChannels[msg.RoomID]; ok {
		hearIt = true
	}
	if _, ok := rc.joinedChannels[msg.RoomID]; ok {
		hearIt = true
	}
	if rc.gbuserMap == nil {
		if _, ok := rc.userMap[msg.User.UserName]; !ok {
			mapUser = true
		}
	}
	rc.RUnlock()
	if mapUser {
		rc.Lock()
		rc.userMap[msg.User.UserName] = msg.User.ID
		rc.Unlock()
	}
	if !hearIt {
		return
	}
	rc.Log(bot.Debug, "DEBUG: Raw incoming msg: %+v", *msg)
	rc.Log(bot.Debug, "DEBUG: Raw incoming user: %+v", *msg.User)
	botMsg := &bot.ConnectorMessage{
		Protocol:      "Rocket",
		UserID:        msg.User.ID,
		UserName:      msg.User.UserName,
		ChannelID:     msg.RoomID,
		ChannelName:   chName,
		MessageText:   msg.Msg,
		MessageObject: msg,
		Client:        rc.rt,
		DirectMessage: directMsg,
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
			if ich.Type == "p" {
				rc.privChannels[ich.ID] = struct{}{}
			}
		}
	}
}

func (rc *rocketConnector) sendMessage(ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	return bot.Ok
}
