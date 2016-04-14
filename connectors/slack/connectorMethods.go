package slack

import (
	"github.com/parsley42/gobot/bot"
)

// SendChannelMessage sends a message to a channel
func (s *slackConnector) SendChannelMessage(c string, m string) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.log(bot.Error, "Channel ID not found for:", c)
		return
	}
	s.conn.SendMessage(s.conn.NewOutgoingMessage(m, chanID))
}

// SendUserMessage sends a direct message to a user
func (s *slackConnector) SendUserMessage(u string, m string) {
	return
}

// SetLogLevel updates the connector log level
func (s *slackConnector) SetLogLevel(l bot.LogLevel) {
	s.Lock()
	s.level = l
	s.Unlock()
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (s *slackConnector) JoinChannel(c string) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.log(bot.Error, "Channel ID not found for:", c)
		return
	}
	_, err := s.api.JoinChannel(chanID)
	if err != nil {
		s.log(bot.Error, "Failed to join channel", c, ":", err, "(try inviting the bot)")
	}
}
