package slack

import (
	"sync"

	"github.com/nlopes/slack"
)

type slackConnector struct {
	api          *slack.Client
	conn         *slack.RTM
	sync.RWMutex                   // for locking connector data structures
	channelIDs   map[string]string // map from channel names to channel IDs
}

func (s *slackConnector) SendChannelMessage(c string, m string) {
	s.conn.SendMessage(s.conn.NewOutgoingMessage(m, c))
}
