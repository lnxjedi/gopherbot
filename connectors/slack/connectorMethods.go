package slack

import (
	"time"

	"github.com/parsley42/gopherbot/bot"
)

// How long to delay between sending bits of a chopped up message
const msgDelay = 200 * time.Millisecond

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (s *slackConnector) GetProtocolUserAttribute(u, attr string) (value string, ok bool) {
	user, ok := s.getUser(u)
	if !ok {
		return "", false
	}
	switch attr {
	case "email":
		return user.Profile.Email, ok
	case "realName", "fullName":
		return user.RealName, ok
	case "firstName":
		return user.Profile.FirstName, ok
	case "lastName":
		return user.Profile.LastName, ok
	case "phone":
		return user.Profile.Phone, ok
	// that's all the attributes slack knows about
	default:
		return "", false
	}
}

func (s *slackConnector) sendMessages(msgs []string, chanID string) {
	l := len(msgs)
	for i, msg := range msgs {
		s.conn.SendMessage(s.conn.NewOutgoingMessage(msg, chanID))
		if i != (l - 1) {
			time.Sleep(msgDelay)
		}
	}
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolChannelMessage(ch string, msg string, f bot.MessageFormat) {
	chanID, ok := s.chanID(ch)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", ch)
		return
	}
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, chanID)
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolUserChannelMessage(u, ch, msg string, f bot.MessageFormat) {
	chanID, ok := s.chanID(ch)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", ch)
		return
	}
	msg = "@" + u + ": " + msg
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, chanID)
}

// SendProtocolUserMessage sends a direct message to a user
func (s *slackConnector) SendProtocolUserMessage(u string, msg string, f bot.MessageFormat) {
	userID, ok := s.userID(u)
	if !ok {
		s.Log(bot.Error, "No user ID found for user:", u)
	}
	var userIMchan string
	var err error
	userIMchan, ok = s.userIMID(userID)
	if !ok {
		s.Log(bot.Warn, "No IM channel found for user:", u, "ID:", userID, "trying to open IM")
		_, _, userIMchan, err = s.conn.OpenIMChannel(userID)
		if err != nil {
			s.Log(bot.Error, "Unable to open an IM channel to user:", u, "ID:", userID)
			return
		}
	}
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, userIMchan)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (s *slackConnector) JoinChannel(c string) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", c)
		return
	}
	_, err := s.api.JoinChannel(chanID)
	if err != nil {
		s.Log(bot.Error, "Failed to join channel", c, ":", err, "(try inviting the bot)")
	}
}
