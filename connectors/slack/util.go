package slack

/* util has most of the struct, type, and const definitions, as well as
most of the internal methods. */

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/nlopes/slack"
	"github.com/parsley42/gobot/bot"
)

const optimeout = 1 * time.Minute

// slackConnector holds all the relevant data about a connection
type slackConnector struct {
	api          *slack.Client
	conn         *slack.RTM
	botName      string                // human-readable name of bot
	botID        string                // slack internal bot ID
	bot.Handler                        // bot API for connectors
	sync.RWMutex                       // shared mutex for locking connector data structures
	channelToID  map[string]string     // map from channel names to channel IDs
	idToChannel  map[string]string     // map from channel ID to channel name
	userInfo     map[string]slack.User // map from user names to slack.User struct
	idToUser     map[string]string     // map from user ID to user name
	userIDToIM   map[string]string     // map from user ID to IM channel ID
	imToUser     map[string]string     // map from IM channel ID to user name
}

// addMessageMentions replaces @username with the slack-internal representation
func (s *slackConnector) addMessageMentions(msg string) string {
	re := regexp.MustCompile(`@[0-9a-z]{1,21}\b`)
	return re.ReplaceAllStringFunc(msg, func(str string) string {
		replace, ok := s.userID(str[1:])
		if ok {
			return "<@" + replace + ">"
		}
		return str
	})
}

// processMessage examines incoming messages and routes them to the appropriate bot
// method.
func (s *slackConnector) processMessage(msg *slack.MessageEvent) {
	s.Log(bot.Trace, fmt.Sprintf("Message received: %v\n", msg))
	re := regexp.MustCompile(`<@U[A-Z0-9]{8}>`) // match a @user mention
	text := msg.Msg.Text
	chanID := msg.Msg.Channel
	mentions := re.FindAllString(text, -1)
	if len(mentions) != 0 {
		mset := make(map[string]bool)
		for _, mention := range mentions {
			mset[mention] = true
		}
		for mention, _ := range mset {
			mID := mention[2:11]
			replace, ok := s.userName(mID)
			if !ok {
				s.Log(bot.Warn, "Couldn't find username for mentioned", mID)
				continue
			}
			text = strings.Replace(text, mention, "@"+replace, -1)
		}
	}
	switch chanID[:1] {
	case "D":
		userName, ok := s.imUser(chanID)
		if !ok {
			s.Log(bot.Warn, "Couldn't find user name for IM", chanID)
			s.DirectMsg(chanID, text)
			return
		}
		s.DirectMsg(userName, text)
	case "C":
		channelName, ok := s.channelName(chanID)
		if !ok {
			s.Log(bot.Warn, "Coudln't find channel name for ID", chanID)
			s.ChannelMsg(chanID, text)
			return
		}
		s.ChannelMsg(channelName, text)
	}
}

// update maps is called whenever there are any changes
// to users or channels, so that plugins can use
// human-readable names for users and channels.
func (s *slackConnector) updateMaps() {
	s.Log(bot.Trace, "Updating maps")
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
	userMap := make(map[string]slack.User)
	userIDMap := make(map[string]string)
	for _, user := range userlist {
		s.Log(bot.Trace, "Mapping user name", user.Name, "to", user.ID)
		userMap[user.Name] = user
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
		s.Log(bot.Trace, "Mapping user ID", userIM.User, "to IM channel", userIM.ID)
		userIMMap[userIM.User] = userIM.ID
		s.Log(bot.Trace, "Mapping IM channel", userIM.ID, "to user name", userIDMap[userIM.User])
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
	chanIDMap := make(map[string]string)
	for _, channel := range chanlist {
		s.Log(bot.Trace, "Mapping channel name", channel.Name, "to", channel.ID)
		chanMap[channel.Name] = channel.ID
		chanIDMap[channel.ID] = channel.Name
	}

	s.Lock()
	s.userInfo = userMap
	s.idToUser = userIDMap
	s.userIDToIM = userIMMap
	s.channelToID = chanMap
	s.idToChannel = chanIDMap
	s.imToUser = userNameMap
	s.Unlock()
	s.Log(bot.Info, "Users updated")
}

func (s *slackConnector) getUser(u string) (user slack.User, ok bool) {
	s.RLock()
	user, ok = s.userInfo[u]
	s.RUnlock()
	if !ok {
		return user, false
	}
	return user, ok
}

func (s *slackConnector) userID(u string) (i string, ok bool) {
	s.RLock()
	user, ok := s.userInfo[u]
	s.RUnlock()
	if !ok {
		return "", false
	}
	return user.ID, ok
}

func (s *slackConnector) userName(i string) (u string, ok bool) {
	s.RLock()
	u, ok = s.idToUser[i]
	s.RUnlock()
	return u, ok
}

func (s *slackConnector) userIMID(u string) (i string, ok bool) {
	s.RLock()
	i, ok = s.userIDToIM[u]
	s.RUnlock()
	return i, ok
}

func (s *slackConnector) chanID(c string) (i string, ok bool) {
	s.RLock()
	i, ok = s.channelToID[c]
	s.RUnlock()
	return i, ok
}

func (s *slackConnector) channelName(i string) (c string, ok bool) {
	s.RLock()
	c, ok = s.idToChannel[i]
	s.RUnlock()
	return c, ok
}

func (s *slackConnector) imUser(c string) (u string, ok bool) {
	s.RLock()
	u, ok = s.imToUser[c]
	s.RUnlock()
	return u, ok
}
