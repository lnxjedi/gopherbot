package slack

import (
	"github.com/parsley42/gopherbot/bot"
)

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
	case "realName":
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

// SendChannelMessage sends a message to a channel
func (s *slackConnector) SendChannelMessage(c string, m string) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", c)
		return
	}
	m = s.addMessageMentions(m)
	s.conn.SendMessage(s.conn.NewOutgoingMessage(m, chanID))
}

// SendUserMessage sends a direct message to a user
func (s *slackConnector) SendUserMessage(u string, m string) {
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
	m = s.addMessageMentions(m)
	s.conn.SendMessage(s.conn.NewOutgoingMessage(m, userIMchan))
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
