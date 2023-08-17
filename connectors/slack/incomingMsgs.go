package slack

import (
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// processMessageSocketMode examines incoming messages, removes extra slack cruft, and
// routes them to the appropriate bot method.
func (s *slackConnector) processMessageSocketMode(msg *slackevents.MessageEvent) {
	s.Log(robot.Trace, "Message received: %v", msg)

	// Channel is always part of the root message; if subtype is
	// message_changed, text and user are part of the submessage
	chanID := msg.Channel
	var userID string
	timestamp := time.Now()
	var message slackevents.MessageEvent
	ci, ok := s.getChannelInfo(chanID)
	if !ok {
		s.Log(robot.Error, "Couldn't find channel info for channel ID", chanID)
		return
	}
	if msg.SubType == "message_changed" {
		message = *msg.Message
		userID = message.User
		if userID == "" {
			if message.BotID != "" {
				userID = message.BotID
			}
		}
		lastlookup := userlast{userID, chanID}
		lastmsgtime.Lock()
		msgtime, exists := lastmsgtime.m[lastlookup]
		lastmsgtime.Unlock()
		if exists && timestamp.Sub(msgtime) < ignorewindow {
			s.Log(robot.Debug, "Ignoring edited message \"%s\" arriving within the ignorewindow: %v", msg.Message.Text, ignorewindow)
			return
		}
		s.Log(robot.Debug, "SubMessage (edited message) received: %v", message)
	} else if msg.SubType == "message_deleted" {
		s.Log(robot.Debug, "Ignoring deleted message in channel '%s'", chanID)
		return
	} else if len(msg.SubType) > 0 && !validSubtype(msg.SubType) {
		s.Log(robot.Warn, "Ignoring message with unknown/unhandled subtype '%s'", msg.SubType)
		return
	} else {
		message = *msg
		userID = message.User
		if len(userID) == 0 {
			if message.BotID != "" {
				userID = message.BotID
			} else if ci.IsIM {
				userID, _ = s.imUserID(chanID)
			}
		}
		lastlookup := userlast{userID, chanID}
		lastmsgtime.Lock()
		lastmsgtime.m[lastlookup] = timestamp
		lastmsgtime.Unlock()
	}
	if len(userID) == 0 {
		s.Log(robot.Debug, "Zero-length userID, ignoring message")
		return
	}
	text := message.Text
	messageID := message.TimeStamp
	ts := message.TimeStamp
	tts := message.ThreadTimeStamp
	threadID := tts
	threadedMessage := false
	if len(tts) == 0 {
		threadID = ts
	} else {
		threadedMessage = true
	}
	// some bot messages don't have any text, so check for a fallback
	if text == "" && len(msg.Attachments) > 0 {
		text = msg.Attachments[0].Fallback
	}
	text = s.processText(text)
	botMsg := &robot.ConnectorMessage{
		Protocol:        "slack",
		UserID:          userID,
		ChannelID:       chanID,
		MessageID:       messageID,
		ThreadID:        threadID,
		ThreadedMessage: threadedMessage,
		DirectMessage:   ci.IsIM,
		BotMessage:      false,
		MessageText:     text,
		MessageObject:   msg,
		Client:          s.api,
	}
	userName, ok := s.userName(userID)
	if !ok {
		s.Log(robot.Debug, "Couldn't find user name for user ID", userID)
	} else {
		botMsg.UserName = userName
	}
	if !ci.IsIM {
		botMsg.ChannelName = ci.Name
	}
	if userID == s.botUserID {
		botMsg.SelfMessage = true
		s.Log(robot.Trace, "Forwarding slack return message '%s' from the robot %s/%s", messageID, userName, userID)
	}
	s.IncomingMessage(botMsg)
}

// processSlashCmdSocketMode examines incoming /<foo> messages routed to the robot,
// removes extra slack cruft, and routes them to the appropriate bot method.
func (s *slackConnector) processSlashCmdSocketMode(cmd *slack.SlashCommand) {
	s.Log(robot.Trace, "Slash command received: %+v", cmd)
	chanID := cmd.ChannelID
	userID := cmd.UserID
	ci, ok := s.getChannelInfo(chanID)
	if !ok {
		s.Log(robot.Error, "Couldn't find channel info for channel ID", chanID)
		return
	}
	text := s.processText(cmd.Text)
	botMsg := &robot.ConnectorMessage{
		Protocol:  "slack",
		UserID:    userID,
		ChannelID: chanID,
		// ThreadID should be empty, and ThreadedMessage always false
		DirectMessage: ci.IsIM,
		BotMessage:    true,
		HiddenMessage: true,
		MessageText:   text,
		MessageObject: cmd,
		Client:        s.api,
	}
	userName, ok := s.userName(userID)
	if !ok {
		s.Log(robot.Debug, "Couldn't find user name for user ID", userID)
	} else {
		botMsg.UserName = userName
	}
	if !ci.IsIM {
		botMsg.ChannelName = ci.Name
	}
	s.IncomingMessage(botMsg)
}

// processMessageRTM examines incoming messages, removes extra slack cruft, and
// routes them to the appropriate bot method.
func (s *slackConnector) processMessageRTM(msg *slack.MessageEvent) {
	s.Log(robot.Trace, "Message received: %v", msg.Msg)

	// Channel is always part of the root message; if subtype is
	// message_changed, text and user are part of the submessage
	chanID := msg.Channel
	var userID string
	timestamp := time.Now()
	var message slack.Msg
	ci, ok := s.getChannelInfo(chanID)
	if !ok {
		s.Log(robot.Error, "Couldn't find channel info for channel ID", chanID)
		return
	}
	if msg.Msg.SubType == "message_changed" {
		message = *msg.SubMessage
		userID = message.User
		if userID == "" {
			if message.BotID != "" {
				userID = message.BotID
			}
		}
		lastlookup := userlast{userID, chanID}
		lastmsgtime.Lock()
		msgtime, exists := lastmsgtime.m[lastlookup]
		lastmsgtime.Unlock()
		if exists && timestamp.Sub(msgtime) < ignorewindow {
			s.Log(robot.Debug, "Ignoring edited message \"%s\" arriving within the ignorewindow: %v", msg.SubMessage.Text, ignorewindow)
			return
		}
		s.Log(robot.Debug, "SubMessage (edited message) received: %v", message)
	} else if msg.Msg.SubType == "message_deleted" {
		s.Log(robot.Debug, "Ignoring deleted message in channel '%s'", chanID)
		return
	} else {
		message = msg.Msg
		userID = message.User
		if len(userID) == 0 {
			if message.BotID != "" {
				userID = message.BotID
			} else if ci.IsIM {
				userID, _ = s.imUserID(chanID)
			}
		}
		lastlookup := userlast{userID, chanID}
		lastmsgtime.Lock()
		lastmsgtime.m[lastlookup] = timestamp
		lastmsgtime.Unlock()
	}
	if len(userID) == 0 {
		s.Log(robot.Debug, "Zero-length userID, ignoring message")
		return
	}
	messageID := msg.Timestamp
	text := message.Text
	// some bot messages don't have any text, so check for a fallback
	if text == "" && len(msg.Attachments) > 0 {
		text = msg.Attachments[0].Fallback
	}
	text = s.processText(text)
	ts := msg.Timestamp
	tts := msg.ThreadTimestamp
	threadID := tts
	threadedMessage := false
	if len(tts) == 0 {
		threadID = ts
	} else {
		threadedMessage = true
	}
	botMsg := &robot.ConnectorMessage{
		Protocol:        "slack",
		UserID:          userID,
		ChannelID:       chanID,
		MessageID:       messageID,
		ThreadID:        threadID,
		ThreadedMessage: threadedMessage,
		DirectMessage:   ci.IsIM,
		BotMessage:      false,
		MessageText:     text,
		MessageObject:   msg,
		Client:          s.api,
	}
	userName, ok := s.userName(userID)
	if !ok {
		s.Log(robot.Debug, "Couldn't find user name for user ID", userID)
	} else {
		botMsg.UserName = userName
	}
	if !ci.IsIM {
		botMsg.ChannelName = ci.Name
	}
	if userID == s.botUserID {
		botMsg.SelfMessage = true
		s.Log(robot.Trace, "Forwarding slack return message '%s' from the robot %s/%s", messageID, userName, userID)
	}
	s.IncomingMessage(botMsg)
}
