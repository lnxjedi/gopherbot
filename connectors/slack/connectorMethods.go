package slack

import (
	"time"

	"github.com/uva-its/gopherbot/bot"
)

// Message send delay; slack has problems with scrolling if messages fly out
// too fast.
const typingDelay = 200 * time.Millisecond
const msgDelay = time.Second

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (s *slackConnector) GetProtocolUserAttribute(u, attr string) (value string, ret bot.RetVal) {
	user, ok := s.getUser(u)
	if !ok {
		return "", bot.UserNotFound
	}
	switch attr {
	case "email":
		return user.Profile.Email, bot.Ok
	case "internalID":
		return user.ID, bot.Ok
	case "realName", "fullName":
		return user.RealName, bot.Ok
	case "firstName":
		return user.Profile.FirstName, bot.Ok
	case "lastName":
		return user.Profile.LastName, bot.Ok
	case "phone":
		return user.Profile.Phone, bot.Ok
	// that's all the attributes we can currently get from slack
	default:
		return "", bot.AttributeNotFound
	}
}

type sendMessage struct {
	message, channel string
}

var messages = make(chan *sendMessage)

func (s *slackConnector) startSendLoop() {
	for {
		send := <-messages
		time.Sleep(typingDelay / 2)
		s.conn.SendMessage(s.conn.NewTypingMessage(send.channel))
		time.Sleep(2 * typingDelay)
		s.conn.SendMessage(s.conn.NewOutgoingMessage(send.message, send.channel))
		time.Sleep(msgDelay)
	}
}

func (s *slackConnector) sendMessages(msgs []string, chanID string) {
	for _, msg := range msgs {
		messages <- &sendMessage{
			message: msg,
			channel: chanID,
		}
	}
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolChannelMessage(ch string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	chanID, ok := s.chanID(ch)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", ch)
		return bot.ChannelNotFound
	}
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, chanID)
	return
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolUserChannelMessage(u, ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	chanID, ok := s.chanID(ch)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", ch)
		ret = bot.ChannelNotFound
	} else if _, ok := s.userID(u); !ok {
		ret = bot.UserNotFound
	}
	if ret != bot.Ok {
		return
	}
	msg = "@" + u + ": " + msg
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, chanID)
	return
}

// SendProtocolUserMessage sends a direct message to a user
func (s *slackConnector) SendProtocolUserMessage(u string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	userID, ok := s.userID(u)
	if !ok {
		s.Log(bot.Error, "No user ID found for user:", u)
		ret = bot.UserNotFound
	}
	var userIMchan string
	var err error
	userIMchan, ok = s.userIMID(userID)
	if !ok {
		s.Log(bot.Warn, "No IM channel found for user:", u, "ID:", userID, "trying to open IM")
		_, _, userIMchan, err = s.conn.OpenIMChannel(userID)
		if err != nil {
			s.Log(bot.Error, "Unable to open an IM channel to user:", u, "ID:", userID)
			ret = bot.FailedUserDM
		}
	}
	if ret != bot.Ok {
		return
	}
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, userIMchan)
	return bot.Ok
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (s *slackConnector) JoinChannel(c string) (ret bot.RetVal) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", c)
		return bot.ChannelNotFound
	}
	_, err := s.api.JoinChannel(chanID)
	if err != nil {
		s.Log(bot.Error, "Failed to join channel", c, ":", err, "(try inviting the bot)")
		return bot.FailedChannelJoin
	}
	return bot.Ok
}
