package slack

/* util has most of the struct, type, and const definitions, as well as
most of the internal methods. */

import (
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/nlopes/slack"
)

const optimeout = 1 * time.Minute

// slackConnector holds all the relevant data about a connection
type slackConnector struct {
	api             *slack.Client
	conn            *slack.RTM
	maxMessageSplit int                       // The maximum # of ~4000 byte messages to send before truncating
	running         bool                      // set on call to Run
	botName         string                    // human-readable name of bot
	botFullName     string                    // human-readble full name of the bot
	botID           string                    // slack internal bot ID
	name            string                    // name for this connector
	teamID          string                    // Slack unique Team ID, for identifying team users
	robot.Handler                             // bot API for connectors
	sync.RWMutex                              // shared mutex for locking connector data structures
	channelInfo     map[string]*slack.Channel // info about all the channels the robot knows about
	channelToID     map[string]string         // map from channel names to channel IDs
	idToChannel     map[string]string         // map from channel ID to channel name
	userIDInfo      map[string]*slack.User    // map from user ID to slack.User
	botUserMap      map[string]string         // gopherbot-engine provided mappings of username to userID
	userMap         map[string]string         // map from user name to user ID
	userIDToIM      map[string]string         // map from user ID to IM channel ID
	imToUserID      map[string]string         // map from IM channel ID to user ID
}

// updateUserList gets an updated list of users from Slack and creates
// maps from userID to *slack.User and from user name to userID. It can also
// be called from userID and userName lookup functions; "u:ID" means return
// the user name for the given ID, "i:user" means return the ID for the given
// user name.
func (s *slackConnector) updateUserList(want string) (ret string) {
	deadline := time.Now().Add(optimeout)
	var (
		err      error
		userlist []slack.User
	)

	userMap := make(map[string]string)
	var botUserMap map[string]string
	s.RLock()
	if s.botUserMap != nil {
		botUserMap = s.botUserMap
	}
	s.RUnlock()
	if botUserMap != nil {
		for name, id := range botUserMap {
			userMap[name] = id
		}
	}
	userIDInfo := make(map[string]*slack.User)
	for tries := uint(0); time.Now().Before(deadline); tries++ {
		// TODO: Check GetUsers - do we need to worry about paging?
		userlist, err = s.api.GetUsers()
		if err == nil {
			break
		}
	}
	if err != nil {
		s.Log(robot.Error, "Protocol timeout updating users: %v\n", err)
	}
	for i, user := range userlist {
		if user.TeamID == s.teamID {
			userIDInfo[user.ID] = &userlist[i]
			if _, ok := userMap[user.Name]; !ok {
				userMap[user.Name] = user.ID
			}
		}
	}
	w := strings.Split(want, ":")
	t := w[0]
	switch t {
	case "i": // want user ID
		u := w[1]
		if r, ok := userMap[u]; ok {
			ret = r
		} else {
			return "" // see above
		}
	case "u": // want user name
		i := w[1]
		if r, ok := userIDInfo[i]; ok {
			ret = r.Name
		} else {
			// Don't update maps on failed lookup, to avoid thrashing
			// locks on repeated lookups of non-users
			return ""
		}
	}
	s.Lock()
	s.userIDInfo = userIDInfo
	s.userMap = userMap
	s.Unlock()
	s.Log(robot.Debug, "User maps updated")
	return
}

func (s *slackConnector) userID(u string) (i string, ok bool) {
	s.RLock()
	userID, ok := s.userMap[u]
	s.RUnlock()
	if !ok {
		i := s.updateUserList("i:" + u)
		if len(i) > 0 {
			return i, true
		}
		s.Log(robot.Error, "Failed ID lookup for user '%s'", u)
		return "", false
	}
	return userID, ok
}

// Eventually Slack won't know what usernames are anymore - but
// this function will still be needed by processMessage to resolve
// mentions of the robot to @botname, since the connector should
// always know the robot's name.
func (s *slackConnector) userName(i string) (user string, found bool) {
	s.RLock()
	if i == s.botID {
		name := s.botName
		s.RUnlock()
		return name, true
	}
	if strings.HasPrefix(i, "B") {
		s.RUnlock()
		return
	}
	var ui *slack.User
	ui, found = s.userIDInfo[i]
	s.RUnlock()
	if found {
		user = ui.Name
	} else {
		u := s.updateUserList("u:" + i)
		if len(u) == 0 {
			s.Log(robot.Error, "Failed username lookup for ID '%s'", i)
			return "", false
		}
		user = u
		found = true
	}
	return
}

func (s *slackConnector) updateChannelMaps(want string) (ret string) {
	var (
		err    error
		cursor string
	)
	limit := 100

	deadline := time.Now().Add(optimeout)

	channelList := make([]slack.Channel, 0)
pageLoop:
	for {
		for tries := uint(0); time.Now().Before(deadline); tries++ {
			var cl []slack.Channel
			params := &slack.GetConversationsParameters{
				Cursor:          cursor,
				ExcludeArchived: "true",
				Limit:           limit,
				Types: []string{
					"public_channel",
					"private_channel",
					"mpim",
					"im",
				},
			}
			cl, cursor, err = s.api.GetConversations(params)
			if len(cl) > 0 {
				channelList = append(channelList, cl...)
			}
			if err == nil && len(cursor) == 0 {
				break pageLoop
			}
			if len(cursor) > 0 {
				deadline = time.Now().Add(optimeout)
			}
		}
		if err != nil {
			s.Log(robot.Error, "Protocol timeout updating channels: %v\n", err)
			break
		}
	}
	userIMMap := make(map[string]string)
	userIMIDMap := make(map[string]string)
	chanMap := make(map[string]string)
	chanIDMap := make(map[string]string)
	chanInfo := make(map[string]*slack.Channel)
	for i, channel := range channelList {
		chanInfo[channel.ID] = &channelList[i]
		if channel.IsIM {
			userIMMap[channel.User] = channel.ID
			userIMIDMap[channel.ID] = channel.User
		} else {
			chanMap[channel.Name] = channel.ID
			chanIDMap[channel.ID] = channel.Name
		}
	}
	w := strings.Split(want, ":")
	t := w[0]
	switch t {
	case "di":
		c := w[1]
		if r, ok := userIMIDMap[c]; ok {
			ret = r
		} else {
			// Don't update maps on failed lookup, to avoid thrashing
			// locks on repeated lookups of non-users
			return ""
		}
	case "dc":
		i := w[1]
		if r, ok := userIMMap[i]; ok {
			ret = r
		} else {
			return "" // see above
		}
	case "ci":
		c := w[1]
		if r, ok := chanMap[c]; ok {
			ret = r
		} else {
			return ""
		}
	case "cc":
		i := w[1]
		if r, ok := chanIDMap[i]; ok {
			ret = r
		} else {
			return ""
		}
	}
	s.Lock()
	s.channelInfo = chanInfo
	s.userIDToIM = userIMMap
	s.imToUserID = userIMIDMap
	s.channelToID = chanMap
	s.idToChannel = chanIDMap
	s.Unlock()
	s.Log(robot.Debug, "Channel maps updated")
	return
}

func (s *slackConnector) getChannelInfo(i string) (c *slack.Channel, ok bool) {
	s.RLock()
	c, ok = s.channelInfo[i]
	s.RUnlock()
	if !ok {
		s.updateChannelMaps("")
		s.RLock()
		c, ok = s.channelInfo[i]
		s.RUnlock()
		if !ok {
			s.Log(robot.Error, "Failed lookup of channel info from ID: %s", i)
			return nil, false
		}
	}
	return c, ok
}

// Get IM conversation from user ID
func (s *slackConnector) userIMID(i string) (c string, ok bool) {
	s.RLock()
	c, ok = s.userIDToIM[i]
	s.RUnlock()
	if !ok {
		c = s.updateChannelMaps("dc:" + i)
		if len(i) == 0 {
			s.Log(robot.Error, "Failed lookup of conversation from user ID: %s", i)
			return "", false
		}
	}
	return i, ok
}

// Get user ID from IM conversation
func (s *slackConnector) imUserID(c string) (i string, found bool) {
	s.RLock()
	i, found = s.imToUserID[c]
	s.RUnlock()
	if !found {
		i = s.updateChannelMaps("di:" + c)
		if len(i) == 0 {
			s.Log(robot.Error, "Failed lookup of user ID from IM: %s", c)
			return "", false
		}
		found = true
	}
	return
}

func (s *slackConnector) chanID(c string) (i string, ok bool) {
	s.RLock()
	i, ok = s.channelToID[c]
	s.RUnlock()
	if !ok {
		c = s.updateChannelMaps("ci:" + c)
		if len(i) == 0 {
			s.Log(robot.Error, "Failed lookup of channel ID for '%s'", c)
			return "", false
		}
	}
	return i, ok
}

func (s *slackConnector) channelName(i string) (c string, ok bool) {
	s.RLock()
	c, ok = s.idToChannel[i]
	s.RUnlock()
	if !ok {
		c = s.updateChannelMaps("cc:" + i)
		if len(i) == 0 {
			s.Log(robot.Error, "Failed lookup of channel name from ID: %s", i)
			return "", false
		}
	}
	return c, ok
}
