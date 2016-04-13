package slack

/* util has most of the struct, type, and const definitions, as well as
most of the internal methods. */

import (
	"log"
	"sync"
	"time"

	"github.com/nlopes/slack"
	bot "github.com/parsley42/gobot/bot"
)

const optimeout = 1 * time.Minute

type slackConnector struct {
	api          *slack.Client
	conn         *slack.RTM
	sync.RWMutex                   // shared mutex for locking connector data structures
	channelIDs   map[string]string // map from channel names to channel IDs
	level        bot.LogLevel      // current log level
}

// log logs messages whenever the connector log level is
// less than the given level
func (s *slackConnector) log(l bot.LogLevel, v ...interface{}) {
	if l >= s.level {
		log.Println(v)
	}
}

// update channels is called whenever there are any channel
// updates, to re-populate the name -> id channel map
func (s *slackConnector) updateChannels() {
	s.log(bot.Trace, "Updating channels")
	deadline := time.Now().Add(optimeout)
	var (
		err      error
		chanlist []slack.Channel
	)
	for tries := uint(0); time.Now().Before(deadline); tries++ {
		chanlist, err = s.api.GetChannels(true)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Fatalf("Protocol timeout updating channels: %v\n", err)
	}
	chanMap := make(map[string]string)
	for _, channel := range chanlist {
		s.log(bot.Trace, "Mapping ", channel.Name, " to ", channel.ID)
		chanMap[channel.Name] = channel.ID
	}
	s.Lock()
	s.channelIDs = chanMap
	s.Unlock()
	s.log(bot.Info, "Channels updated")
}

func (s *slackConnector) chanID(c string) (i string) {
	s.Lock()
	i = s.channelIDs[c]
	s.Unlock()
	return i
}
