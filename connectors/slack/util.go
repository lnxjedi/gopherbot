package slack

/* util has most of the struct, type, and const definitions, as well as
most of the internal methods. */

import (
	"log"
	"sync"
	"time"

	"github.com/nlopes/slack"
	"github.com/parsley42/gobot/bot"
)

const optimeout = 1 * time.Minute

type slackConnector struct {
	api          *slack.Client
	conn         *slack.RTM
	sync.RWMutex                   // shared mutex for locking connector data structures
	channelToID  map[string]string // map from channel names to channel IDs
	userToID     map[string]string // map from user names to user IDs
	userIDToIM   map[string]string // map from user ID to IM channel ID
	imToUser     map[string]string // map from IM channel ID to user name
	level        bot.LogLevel      // current log level
}

// log logs messages whenever the connector log level is
// less than the given level
func (s *slackConnector) log(l bot.LogLevel, v ...interface{}) {
	if l >= s.level {
		var prefix string
		switch l {
		case bot.Trace:
			prefix = "Trace:"
		case bot.Debug:
			prefix = "Debug:"
		case bot.Info:
			prefix = "Info"
		case bot.Warn:
			prefix = "Warning:"
		case bot.Error:
			prefix = "Error"
		}
		log.Println(prefix, v)
	}
}

// update maps is called whenever there are any changes
// to users or channels, so that plugins can use
// human-readable names for users and channels.
func (s *slackConnector) updateMaps() {
	s.log(bot.Trace, "Updating maps")
	deadline := time.Now().Add(optimeout)
	var (
		err        error
		userlist   []slack.User
		userIMlist []slack.IM
		chanlist   []slack.Channel
	)

	for tries := uint(0); time.Now().Before(deadline); tries++ {
		userlist, err = s.api.GetUsers()
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Fatalf("Protocol timeout updating users: %v\n", err)
	}
	userMap := make(map[string]string)
	userIDMap := make(map[string]string)
	for _, user := range userlist {
		s.log(bot.Trace, "Mapping user name", user.Name, "to", user.ID)
		userMap[user.Name] = user.ID
		userIDMap[user.ID] = user.Name
	}

	for tries := uint(0); time.Now().Before(deadline); tries++ {
		userIMlist, err = s.api.GetIMChannels()
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Fatalf("Protocol timeout updating IMchannels: %v\n", err)
	}
	userIMMap := make(map[string]string)
	userNameMap := make(map[string]string)
	for _, userIM := range userIMlist {
		s.log(bot.Trace, "Mapping user ID", userIM.User, "to IM channel", userIM.ID)
		userIMMap[userIM.User] = userIM.ID
		s.log(bot.Trace, "Mapping IM channel", userIM.ID, "to user name", userIDMap[userIM.User])
		userNameMap[userIM.ID] = userIDMap[userIM.User]
	}

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
		s.log(bot.Trace, "Mapping channel name", channel.Name, "to", channel.ID)
		chanMap[channel.Name] = channel.ID
	}

	s.Lock()
	s.userToID = userMap
	s.userIDToIM = userIMMap
	s.channelToID = chanMap
	s.imToUser = userNameMap
	s.Unlock()
	s.log(bot.Info, "Users updated")
}

func (s *slackConnector) userID(c string) (i string, ok bool) {
	s.RLock()
	i, ok = s.userToID[c]
	s.RUnlock()
	return i, ok
}

func (s *slackConnector) userIMID(c string) (i string, ok bool) {
	s.RLock()
	i, ok = s.userIDToIM[c]
	s.RUnlock()
	return i, ok
}

func (s *slackConnector) chanID(c string) (i string, ok bool) {
	s.RLock()
	i, ok = s.channelToID[c]
	s.RUnlock()
	return i, ok
}

func (s *slackConnector) imUser(c string) (u string, ok bool) {
	s.RLock()
	u, ok = s.imToUser[c]
	s.RUnlock()
	return u, ok
}
