package slack

/* util has most of the struct, type, and const definitions, as well as
most of the internal methods. */

import (
	"bytes"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// Soft hyphen. *shrug*
const escapePad = "\u00AD"

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

func optQuote(msg string, f robot.MessageFormat) string {
	if f == robot.Fixed {
		return "```" + msg + "```"
	}
	return msg
}

var mentionMatch = `[0-9a-z](?:[-_0-9a-z.]{0,19}[_0-9a-z])?`
var mentionRe = regexp.MustCompile(`@` + mentionMatch + `\b`)
var usernameRe = regexp.MustCompile(`^` + mentionMatch + `$`)

// slackifyMessage replaces @username with the slack-internal representation, handles escaping,
// takes care of formatting, and segments the message if needed.
func (s *slackConnector) slackifyMessage(prefix, msg string, f robot.MessageFormat) []string {
	maxSize := slack.MaxMessageTextLength - 500 // workaround for large message disconnects
	if f == robot.Fixed {
		maxSize -= 6
	}
	sbytes := []byte(msg)
	// 'escape' special chars; NOTE: this should be covered by slack.MsgOptions now.
	// if f == robot.Variable {
	// }

	// Eventually, this will only work for users configured in the
	// UserRoster from robot.yaml
	if f == robot.Raw {
		sbytes = mentionRe.ReplaceAllFunc(sbytes, func(bytes []byte) []byte {
			mentioned := string(bytes[1:])
			switch mentioned {
			case "here", "channel", "everyone":
				return []byte("<!" + mentioned + ">")
			}
			replace, ok := s.userID(string(bytes[1:]), true)
			if ok {
				return []byte("<@" + replace + ">")
			}
			return bytes
		})
	} else {
		sbytes = bytes.Replace(sbytes, []byte("&"), []byte("&amp;"), -1)
		sbytes = bytes.Replace(sbytes, []byte("<"), []byte("&lt;"), -1)
		sbytes = bytes.Replace(sbytes, []byte(">"), []byte("&gt;"), -1)
	}
	if f == robot.Variable {
		// 'escape' special chars that aren't covered by disabling markdown.
		for _, padChar := range []string{"`", "*", "_", ":"} {
			padBytes := []byte(padChar)
			paddedBytes := []byte(escapePad + padChar)
			sbytes = bytes.Replace(sbytes, padBytes, paddedBytes, -1)
		}
	}
	if len(prefix) > 0 {
		sbytes = append([]byte(prefix), sbytes...)
	}
	msgLen := len(sbytes)
	if msgLen <= maxSize {
		return []string{optQuote(string(sbytes), f)}
	}
	// It's too big, gotta chop it up. We will send at most maxMessageSplit
	// messages, plus "(message truncated)".
	msgs := make([]string, 0, s.maxMessageSplit+1)
	s.Log(robot.Info, "Message too long, segmenting: %d bytes", msgLen)
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

var reAddedLinks = regexp.MustCompile(`<https?://[\w-./]+\|([\w-./]+)>`) // match a slack-inserted link
var reLinks = regexp.MustCompile(`<(https?://[.\w-:/?=~]+)>`)            // match a link where slack added <>
var reUser = regexp.MustCompile(`<@U[A-Z0-9]{7,21}>`)                    // match a @user mention
var reMailToLink = regexp.MustCompile(`<mailto:[^|]+\|([\w-./@]+)>`)     // match mailto links

// I don't love this: if the message text is '<foo>', the robot sees '&lt;foo&gt;'. HOWEVER,
// if the message text is '&lt;foo&gt;', the robot STILL sees '&lt;foo&gt;'. Still, '<' and '>'
// are more useful than '&lt;' and '&gt;', so we always send the angle brackets.
var reLeftAngle = regexp.MustCompile(`&lt;`)
var reRightAngle = regexp.MustCompile(`&gt;`)

func (s *slackConnector) processText(text string) string {
	// Remove auto-links - chatbots don't want those
	text = reAddedLinks.ReplaceAllString(text, "$1")
	text = reLinks.ReplaceAllString(text, "$1")
	text = reMailToLink.ReplaceAllString(text, "$1")
	// Convert '&lt;' and '&gt;' to angle brackets.
	text = reLeftAngle.ReplaceAllString(text, "<")
	text = reRightAngle.ReplaceAllString(text, ">")

	mentions := reUser.FindAllString(text, -1)
	if len(mentions) != 0 {
		mset := make(map[string]bool)
		for _, mention := range mentions {
			mset[mention] = true
		}
		for mention := range mset {
			mID := mention[2 : len(mention)-1]
			replace, ok := s.userName(mID)
			if !ok {
				s.Log(robot.Warn, "Couldn't find username for mentioned", mID)
				continue
			}
			text = strings.Replace(text, mention, "@"+replace, -1)
		}
	}
	return text
}

// processMessageSocketMode examines incoming messages, removes extra slack cruft, and
// routes them to the appropriate bot method.
func (s *slackConnector) processMessageSocketMode(msg *slackevents.MessageEvent) {
	s.Log(robot.Trace, "Message received: %v", msg)

	// Channel is always part of the root message; if subtype is
	// message_changed, text and user are part of the submessage
	chanID := msg.Channel
	var userID string
	timestamp := time.Now()
	var message slackevents.MessageEvent
	ci, ok := s.getChannelInfo(chanID)
	if !ok {
		s.Log(robot.Error, "Couldn't find channel info for channel ID", chanID)
		return
	}
	if msg.SubType == "message_changed" {
		message = *msg.Message
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
			s.Log(robot.Debug, "Ignoring edited message \"%s\" arriving within the ignorewindow: %v", msg.Message.Text, ignorewindow)
			return
		}
		s.Log(robot.Debug, "SubMessage (edited message) received: %v", message)
	} else if msg.SubType == "message_deleted" {
		s.Log(robot.Debug, "Ignoring deleted message in channel '%s'", chanID)
		return
	} else {
		message = *msg
		userID = message.User
		if len(userID) == 0 {
			if message.BotID != "" {
				userID = message.BotID
			} else if ci.IsIM {
				userID, _ = s.imUserID(chanID)
			}
		}
		lastlookup := userlast{userID, chanID}
		lastmsgtime.Lock()
		lastmsgtime.m[lastlookup] = timestamp
		lastmsgtime.Unlock()
	}
	if len(userID) == 0 {
		s.Log(robot.Debug, "Zero-length userID, ignoring message")
		return
	}
	text := message.Text
	messageID := message.TimeStamp
	ts := message.TimeStamp
	tts := message.ThreadTimeStamp
	threadID := tts
	threadedMessage := false
	if len(tts) == 0 {
		threadID = ts
	} else {
		threadedMessage = true
	}
	// some bot messages don't have any text, so check for a fallback
	if text == "" && len(msg.Attachments) > 0 {
		text = msg.Attachments[0].Fallback
	}
	text = s.processText(text)
	botMsg := &robot.ConnectorMessage{
		Protocol:        "slack",
		UserID:          userID,
		ChannelID:       chanID,
		MessageID:       messageID,
		ThreadID:        threadID,
		ThreadedMessage: threadedMessage,
		DirectMessage:   ci.IsIM,
		BotMessage:      false,
		MessageText:     text,
		MessageObject:   msg,
		Client:          s.api,
	}
	userName, ok := s.userName(userID)
	if !ok {
		s.Log(robot.Debug, "Couldn't find user name for user ID", userID)
	} else {
		botMsg.UserName = userName
	}
	if !ci.IsIM {
		botMsg.ChannelName = ci.Name
	}
	if userID == s.botUserID {
		botMsg.SelfMessage = true
		s.Log(robot.Trace, "forwarding slack return message '%s' from the robot %s/%s", messageID, userName, userID)
	}
	s.IncomingMessage(botMsg)
}

// processSlashCmdSocketMode examines incoming /<foo> messages routed to the robot,
// removes extra slack cruft, and routes them to the appropriate bot method.
func (s *slackConnector) processSlashCmdSocketMode(cmd *slack.SlashCommand) {
	s.Log(robot.Trace, "slash command received: %+v", cmd)
	chanID := cmd.ChannelID
	userID := cmd.UserID
	ci, ok := s.getChannelInfo(chanID)
	if !ok {
		s.Log(robot.Error, "Couldn't find channel info for channel ID", chanID)
		return
	}
	text := s.processText(cmd.Text)
	botMsg := &robot.ConnectorMessage{
		Protocol:  "slack",
		UserID:    userID,
		ChannelID: chanID,
		// ThreadID should be empty, and ThreadedMessage always false
		DirectMessage: ci.IsIM,
		BotMessage:    true,
		MessageText:   text,
		MessageObject: cmd,
		Client:        s.api,
	}
	userName, ok := s.userName(userID)
	if !ok {
		s.Log(robot.Debug, "Couldn't find user name for user ID", userID)
	} else {
		botMsg.UserName = userName
	}
	if !ci.IsIM {
		botMsg.ChannelName = ci.Name
	}
	s.IncomingMessage(botMsg)
}

// processMessageRTM examines incoming messages, removes extra slack cruft, and
// routes them to the appropriate bot method.
func (s *slackConnector) processMessageRTM(msg *slack.MessageEvent) {
	s.Log(robot.Trace, "Message received: %v", msg.Msg)

	// Channel is always part of the root message; if subtype is
	// message_changed, text and user are part of the submessage
	chanID := msg.Channel
	var userID string
	timestamp := time.Now()
	var message slack.Msg
	ci, ok := s.getChannelInfo(chanID)
	if !ok {
		s.Log(robot.Error, "Couldn't find channel info for channel ID", chanID)
		return
	}
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
			s.Log(robot.Debug, "Ignoring edited message \"%s\" arriving within the ignorewindow: %v", msg.SubMessage.Text, ignorewindow)
			return
		}
		s.Log(robot.Debug, "SubMessage (edited message) received: %v", message)
	} else if msg.Msg.SubType == "message_deleted" {
		s.Log(robot.Debug, "Ignoring deleted message in channel '%s'", chanID)
		return
	} else {
		message = msg.Msg
		userID = message.User
		if len(userID) == 0 {
			if message.BotID != "" {
				userID = message.BotID
			} else if ci.IsIM {
				userID, _ = s.imUserID(chanID)
			}
		}
		lastlookup := userlast{userID, chanID}
		lastmsgtime.Lock()
		lastmsgtime.m[lastlookup] = timestamp
		lastmsgtime.Unlock()
	}
	if len(userID) == 0 {
		s.Log(robot.Debug, "zero-length userID, ignoring message")
		return
	}
	messageID := msg.Timestamp
	text := message.Text
	// some bot messages don't have any text, so check for a fallback
	if text == "" && len(msg.Attachments) > 0 {
		text = msg.Attachments[0].Fallback
	}
	text = s.processText(text)
	ts := msg.Timestamp
	tts := msg.ThreadTimestamp
	threadID := tts
	threadedMessage := false
	if len(tts) == 0 {
		threadID = ts
	} else {
		threadedMessage = true
	}
	botMsg := &robot.ConnectorMessage{
		Protocol:        "slack",
		UserID:          userID,
		ChannelID:       chanID,
		MessageID:       messageID,
		ThreadID:        threadID,
		ThreadedMessage: threadedMessage,
		DirectMessage:   ci.IsIM,
		BotMessage:      false,
		MessageText:     text,
		MessageObject:   msg,
		Client:          s.api,
	}
	userName, ok := s.userName(userID)
	if !ok {
		s.Log(robot.Debug, "Couldn't find user name for user ID", userID)
	} else {
		botMsg.UserName = userName
	}
	if !ci.IsIM {
		botMsg.ChannelName = ci.Name
	}
	if userID == s.botUserID {
		botMsg.SelfMessage = true
		s.Log(robot.Trace, "forwarding slack return message '%s' from the robot %s/%s", messageID, userName, userID)
	}
	s.IncomingMessage(botMsg)
}
