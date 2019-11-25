// Package rocket implements a connector for Rocket.Chat
package rocket

import (
	"crypto/md5"
	"time"

	models "github.com/lnxjedi/gopherbot/connectors/rocket/models"
	"github.com/lnxjedi/gopherbot/robot"
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
		rc.Log(robot.Error, "failed subscribing to '__my_messages__, won't hear messages: %v", err)
	}

	// Detect UserName
	// TODO: is there a better way to get the robot's UserName?
	if sch, err := rc.rt.GetChannelSubscriptions(); err == nil {
		detected := false
		for _, ch := range sch {
			if len(ch.User.UserName) > 0 {
				detected = true
				userName = ch.User.UserName
				rc.SetBotMention(userName)
				break
			}
		}
		if !detected {
			rc.Log(robot.Warn, "unable to detect username from rocket channel subscriptions")
		}
	} else {
		rc.Log(robot.Error, "failed getting rocket channel subscriptions, can't get username: %v", err)
	}

	mstop := make(chan struct{})
	// duplicate messages loop
	go func() {
	mloop:
		for {
			select {
			case <-mstop:
				break mloop
			case mq := <-check:
				m := mq.msgTrack
				_, ok := trackedMsgs[m]
				if !ok {
					rc.Log(robot.Debug, "DEBUG: recording message %s", m.msgID)
					trackedMsgs[m] = time.Now()
				}
				mq.reply <- ok
			case <-time.After(msgExpire / 2):
				now := time.Now()
				for m, t := range trackedMsgs {
					if now.Sub(t) > msgExpire {
						rc.Log(robot.Debug, "DEBUG: expiring message %s", m.msgID)
						delete(trackedMsgs, m)
					}
				}
			}
		}
	}()
loop:
	for {
		select {
		case pmsg := <-incoming:
			rc.processMessage(&pmsg)
		case <-stop:
			rc.Log(robot.Debug, "Received stop in connector")
			break loop
		}
	}
}

// processMessage creates a robot.ConnectorMessage and calls
// robot.IncomingMessage
func (rc *rocketConnector) processMessage(msg *models.Message) {
	if len(msg.Msg) == 0 {
		return
	}
	if msg.User.ID == userID {
		return
	}
	if msg.User.UserName == userName {
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
	if _, ok := rc.userNameIDMap[msg.User.UserName]; !ok {
		mapUser = true
	}
	rc.RUnlock()
	if mapUser {
		// TODO: is there a better way of mapping username to ID?
		rc.Lock()
		rc.userNameIDMap[msg.User.UserName] = msg.User.ID
		rc.userIDNameMap[msg.User.ID] = msg.User.UserName
		rc.Unlock()
	}
	if !hearIt {
		return
	}
	rc.Log(robot.Debug, "DEBUG: Raw incoming msg: %+v", *msg)
	rc.Log(robot.Debug, "DEBUG: Raw incoming user: %+v", *msg.User)
	// Check for and ignore duplicate messages
	mHash := msgQuery{make(chan bool), msgTrack{msg.ID, md5.Sum([]byte(msg.Msg))}}
	if check <- mHash; <-mHash.reply {
		rc.Log(robot.Debug, "DEBUG: ignoring duplicate message %s", msg.ID)
		return
	}
	botMsg := &robot.ConnectorMessage{
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
	inChannels, ierr := rc.rt.GetChannelsIn()
	if ierr != nil {
		rc.Log(robot.Error, "rocket getting channels in: %v", ierr)
		return
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

func formatMessage(msg string, f robot.MessageFormat) string {
	if f == robot.Fixed {
		msg = "```" + msg + "```"
	}
	return msg
}

// sendMessage takes "channel" or "<chanID>" and sends the pre-formatted
// message.
func (rc *rocketConnector) sendMessage(ch, msg string) (ret robot.RetVal) {
	var chanID string
	found := false
	chanID, found = rc.ExtractID(ch)
	if !found {
		rc.RLock()
		chanID, found = rc.channelIDs[ch]
		rc.RUnlock()
	}
	if !found {
		return robot.ChannelNotFound
	}
	sendChan := models.Channel{ID: chanID}
	m := rc.rt.NewMessage(&sendChan, msg)
	if _, err := rc.rt.SendMessage(m); err != nil {
		return robot.FailedMessageSend
	}
	return robot.Ok
}
