package slack

import (
	bot "github.com/parsley42/gobot/bot"
)

// SendChannelMessage sends a message to a channel
func (s *slackConnector) SendChannelMessage(c string, m string) {
	s.conn.SendMessage(s.conn.NewOutgoingMessage(m, s.chanID(c)))
}

// SetLogLevel updates the connector log level
func (s *slackConnector) SetLogLevel(l bot.LogLevel) {
	s.Lock()
	s.level = l
	s.Unlock()
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (s *slackConnector) JoinChannel(c string) {
	_, err := s.api.JoinChannel(s.chanID(c))
	if err != nil {
		s.log(bot.Error, "Failed to join channel", c, ":", err, "(try inviting the bot)")
	}
}
