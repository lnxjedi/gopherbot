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
	channelIDs   map[string]string // map from channel names to channel IDs
	userIDs      map[string]string // map from user names to user IDs
	userIMIDs    map[string]string // map from user ID to IM channel ID
	level        bot.LogLevel      // current log level
}

// log logs messages whenever the connector log level is
// less than the given level
func (s *slackConnector) log(l bot.LogLevel, v ...interface{}) {
	if l >= s.level {
		var prefix string
		switch s.level {
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

// update users is called whenever there are any user
// updates, to re-populate the name -> id user map
func (s *slackConnector) updateUsers() {
	s.log(bot.Trace, "Updating users")
	deadline := time.Now().Add(optimeout)
	var (
		err      error
		userlist []slack.User
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
	for _, user := range userlist {
		s.log(bot.Trace, "Mapping ", user.Name, " to ", user.ID)
		userMap[user.Name] = user.ID
	}
	s.Lock()
	s.userIDs = userMap
	s.Unlock()
	s.log(bot.Info, "Users updated")
}

func (s *slackConnector) userID(c string) (i string, ok bool) {
	s.Lock()
	i, ok = s.userIDs[c]
	s.Unlock()
	return i, ok
}

// update IMchannels is called whenever there are any IM channel
// updates, to re-populate the user id -> channel id map
func (s *slackConnector) updateIMChannels() {
	s.log(bot.Trace, "Updating IM channels")
	deadline := time.Now().Add(optimeout)
	var (
		err        error
		userIMlist []slack.IM
	)
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
	for _, userIM := range userIMlist {
		s.log(bot.Trace, "Mapping ", userIM.User, " to ", userIM.ID)
		userIMMap[userIM.User] = userIM.ID
	}
	s.Lock()
	s.userIMIDs = userIMMap
	s.Unlock()
	s.log(bot.Info, "IMChannels updated")
}

func (s *slackConnector) userIMID(c string) (i string, ok bool) {
	s.Lock()
	i, ok = s.userIMIDs[c]
	s.Unlock()
	return i, ok
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

func (s *slackConnector) chanID(c string) (i string, ok bool) {
	s.Lock()
	i, ok = s.channelIDs[c]
	s.Unlock()
	return i, ok
}
