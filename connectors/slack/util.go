package slack

/* util has most of the struct, type, and const definitions, as well as
most of the internal methods. */

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/nlopes/slack"
)

const optimeout = 1 * time.Minute

// slackConnector holds all the relevant data about a connection
type slackConnector struct {
	api             *slack.Client
	conn            *slack.RTM
	maxMessageSplit int                   // The maximum # of ~4000 byte messages to send before truncating
	running         bool                  // set on call to Run
	botName         string                // human-readable name of bot
	botFullName     string                // human-readble full name of the bot
	botID           string                // slack internal bot ID
	name            string                // name for this connector
	bot.Handler                           // bot API for connectors
	sync.RWMutex                          // shared mutex for locking connector data structures
	channelToID     map[string]string     // map from channel names to channel IDs
	idToChannel     map[string]string     // map from channel ID to channel name
	userInfo        map[string]slack.User // map from user names to slack.User struct
	idToUser        map[string]string     // map from user ID to user name
	userIDToIM      map[string]string     // map from user ID to IM channel ID
	imToUser        map[string]string     // map from IM channel ID to user name
}

type userlast struct {
	user, channel string
}

var lastmsgtime = struct {
	m map[userlast]time.Time
	sync.Mutex
}{
	make(map[userlast]time.Time),
	sync.Mutex{},
}

// If we get back an edited message from a user in a channel within the
// ignorewindow ... well, we ignore it. The problem is, the Slack service will
// on occasion edit a user message, and the robot was seeing this as the user
// sending the same command twice in short order.
const ignorewindow = 2 * time.Second

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

func optQuote(msg string, f bot.MessageFormat) string {
	if f == bot.Fixed {
		return "```" + msg + "```"
	}
	return msg
}

// slackifyMessage replaces @username with the slack-internal representation, handles escaping,
// takes care of formatting, and segments the message if needed.
func (s *slackConnector) slackifyMessage(msg string, f bot.MessageFormat) []string {
	maxSize := slack.MaxMessageTextLength - 500 // workaround for large message disconnects
	if f == bot.Fixed {
		maxSize -= 6
	}
	sbytes := []byte(msg)
	sbytes = bytes.Replace(sbytes, []byte("&"), []byte("&amp;"), -1)
	sbytes = bytes.Replace(sbytes, []byte("<"), []byte("&lt;"), -1)
	sbytes = bytes.Replace(sbytes, []byte(">"), []byte("&gt;"), -1)

	mentionRe := regexp.MustCompile(`@[0-9a-z]{1,21}\b`)
	sbytes = mentionRe.ReplaceAllFunc(sbytes, func(bytes []byte) []byte {
		replace, ok := s.userID(string(bytes[1:]))
		if ok {
			return []byte("<@" + replace + ">")
		}
		return bytes
	})
	msgLen := len(sbytes)
	if msgLen <= maxSize {
		return []string{optQuote(string(sbytes), f)}
	}
	// It's too big, gotta chop it up. We will send at most maxMessageSplit
	// messages, plus "(message truncated)".
	msgs := make([]string, 0, s.maxMessageSplit+1)
	s.Log(bot.Info, fmt.Sprintf("Message too long, segmenting: %d bytes", msgLen))
	// Chop it up into <=maxSize pieces
	for len(sbytes) > maxSize && len(msgs) < s.maxMessageSplit {
		lineEnd := bytes.LastIndexByte(sbytes[:maxSize], byte('\n'))
		if lineEnd == -1 { // no newline in this chunk
			msgs = append(msgs, optQuote(string(sbytes[:maxSize]), f))
			sbytes = sbytes[maxSize:]
		} else {
			msgs = append(msgs, optQuote(string(sbytes[:lineEnd]), f))
			sbytes = sbytes[lineEnd+1:] // skip over the newline
		}
	}
	if len(msgs) == s.maxMessageSplit { // we've maxed out
		if len(sbytes) > 0 { // if there's anything left, we've truncated
			msgs = append(msgs, "(message too long, truncated)")
		}
	} else { // the last chunk fits
		msgs = append(msgs, optQuote(string(sbytes), f))
	}
	return msgs
}

// processMessage examines incoming messages, removes extra slack cruft, and
// routes them to the appropriate bot method.
func (s *slackConnector) processMessage(msg *slack.MessageEvent) {
	s.Log(bot.Trace, fmt.Sprintf("Message received: %v", msg.Msg))

	reAddedLinks := regexp.MustCompile(`<https?://[\w-.]+\|([\w-.]+)>`) // match a slack-inserted link
	reLinks := regexp.MustCompile(`<(https?://[.\w-:/?=~]+)>`)          // match a link where slack added <>
	reUser := regexp.MustCompile(`<@U[A-Z0-9]{8}>`)                     // match a @user mention

	// Channel is always part of the root message; if subtype is
	// message_changed, text and user are part of the submessage
	chanID := msg.Channel
	var userID string
	timestamp := time.Now()
	var message slack.Msg
	if msg.Msg.SubType == "message_changed" {
		message = *msg.SubMessage
		userID = message.User
		if userID == "" {
			if message.BotID != "" {
				userID = message.BotID
			}
		}
		lastlookup := userlast{userID, chanID}
		lastmsgtime.Lock()
		msgtime, exists := lastmsgtime.m[lastlookup]
		lastmsgtime.Unlock()
		if exists && timestamp.Sub(msgtime) < ignorewindow {
			s.Log(bot.Debug, fmt.Sprintf("Ignoring edited message \"%s\" arriving within the ignorewindow: %v", msg.SubMessage.Text, ignorewindow))
			return
		}
		s.Log(bot.Debug, fmt.Sprintf("SubMessage (edited message) received: %v", message))
	} else {
		message = msg.Msg
		userID = message.User
		if userID == "" {
			if message.BotID != "" {
				userID = message.BotID
			}
		}
		lastlookup := userlast{userID, chanID}
		lastmsgtime.Lock()
		lastmsgtime.m[lastlookup] = timestamp
		lastmsgtime.Unlock()
	}
	text := message.Text
	// some bot messages don't have any text, so check for a fallback
	if text == "" && len(msg.Attachments) > 0 {
		text = msg.Attachments[0].Fallback
	}
	// Remove auto-links - chatbots don't want those
	text = reAddedLinks.ReplaceAllString(text, "$1")
	text = reLinks.ReplaceAllString(text, "$1")

	userName, ok := s.userName(userID)
	if !ok {
		s.Log(bot.Error, "Couldn't find user name for user ID", userID)
		userName = userID
	}
	mentions := reUser.FindAllString(text, -1)
	if len(mentions) != 0 {
		mset := make(map[string]bool)
		for _, mention := range mentions {
			mset[mention] = true
		}
		for mention := range mset {
			mID := mention[2:11]
			replace, ok := s.userName(mID)
			if !ok {
				s.Log(bot.Warn, "Couldn't find username for mentioned", mID)
				continue
			}
			text = strings.Replace(text, mention, "@"+replace, -1)
		}
	}
	s.RLock()
	connector := s.name
	s.RUnlock()
	switch chanID[:1] {
	case "D":
		directUserName, ok := s.imUser(chanID)
		if directUserName != userName { // sometimes the bot hears his own last message
			s.Log(bot.Debug, fmt.Sprintf("Direct message user \"%s\" doesn't match sending user \"%s\", ignoring", directUserName, userName))
			return
		}
		if !ok {
			s.Log(bot.Warn, "Couldn't find user name for IM", chanID)
			s.IncomingMessage("", chanID, text, connector, bot.Slack, msg)
			return
		}
		s.IncomingMessage("", directUserName, text, connector, bot.Slack, msg)
	case "C", "G":
		channelName, ok := s.channelName(chanID)
		if !ok {
			s.Log(bot.Warn, "Coudln't find channel name for ID", chanID)
			s.IncomingMessage(chanID, userName, text, connector, bot.Slack, msg)
			return
		}
		s.IncomingMessage(channelName, userName, text, connector, bot.Slack, msg)
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
		grouplist  []slack.Group
	)

	for tries := uint(0); time.Now().Before(deadline); tries++ {
		userlist, err = s.api.GetUsers()
		if err == nil {
			break
		}
	}
	if err != nil {
		s.Log(bot.Fatal, fmt.Sprintf("Protocol timeout updating users: %v\n", err))
	}
	userMap := make(map[string]slack.User)
	userIDMap := make(map[string]string)
	for _, user := range userlist {
		s.Log(bot.Trace, "Mapping user name", user.Name, "to", user.ID)
		userMap[user.Name] = user
		userIDMap[user.ID] = user.Name
	}
	for botID, botName := range bots {
		s.Log(bot.Trace, "Mapping bot ID", botID, "to name:", botName)
		// note that we don't map the name to anything, since you can't (currently?)
		// speak to a bot/app
		userIDMap[botID] = botName
	}

	for tries := uint(0); time.Now().Before(deadline); tries++ {
		userIMlist, err = s.api.GetIMChannels()
		if err == nil {
			break
		}
	}
	if err != nil {
		s.Log(bot.Fatal, fmt.Sprintf("Protocol timeout updating IMchannels: %v\n", err))
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
		s.Log(bot.Fatal, fmt.Sprintf("Protocol timeout updating channels: %v\n", err))
	}
	for tries := uint(0); time.Now().Before(deadline); tries++ {
		grouplist, err = s.api.GetGroups(true)
		if err == nil {
			break
		}
	}
	if err != nil {
		s.Log(bot.Fatal, fmt.Sprintf("Protocol timeout updating groups: %v\n", err))
	}
	chanMap := make(map[string]string)
	chanIDMap := make(map[string]string)
	for _, channel := range chanlist {
		s.Log(bot.Trace, "Mapping channel name", channel.Name, "to channel", channel.ID)
		chanMap[channel.Name] = channel.ID
		chanIDMap[channel.ID] = channel.Name
	}
	for _, group := range grouplist {
		s.Log(bot.Trace, "Mapping channel name", group.Name, "to group", group.ID)
		chanMap[group.Name] = group.ID
		chanIDMap[group.ID] = group.Name
	}

	s.Lock()
	s.userInfo = userMap
	s.idToUser = userIDMap
	s.userIDToIM = userIMMap
	s.channelToID = chanMap
	s.idToChannel = chanIDMap
	s.imToUser = userNameMap
	s.Unlock()
	s.Log(bot.Info, "User/Group/Channel maps updated")
}
