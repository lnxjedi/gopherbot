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

const escapePad = "\f"

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
	// 'escape' special chars
	if f == bot.Variable {
		for _, padChar := range []string{"`", "*", "_", "@", "#", ":"} {
			padBytes := []byte(padChar)
			paddedBytes := []byte(escapePad + padChar + escapePad)
			sbytes = bytes.Replace(sbytes, padBytes, paddedBytes, -1)
		}
	}

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

	reAddedLinks := regexp.MustCompile(`<https?://[\w-./]+\|([\w-./]+)>`) // match a slack-inserted link
	reLinks := regexp.MustCompile(`<(https?://[.\w-:/?=~]+)>`)            // match a link where slack added <>
	reUser := regexp.MustCompile(`<@U[A-Z0-9]{8}>`)                       // match a @user mention

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
	ci, ok := s.getChannelInfo(chanID)
	if !ok {
		s.Log(bot.Error, "Couldn't find channel info for channel ID", chanID)
		return
	}
	if ci.IsIM {
		directUserName, ok := s.imUser(chanID)
		if directUserName != userName { // sometimes the bot hears his own last message
			s.Log(bot.Debug, fmt.Sprintf("Direct message user \"%s\" doesn't match sending user \"%s\", ignoring", directUserName, userName))
			return
		}
		if !ok {
			s.Log(bot.Warn, "Couldn't find user name for IM", chanID)
			s.IncomingMessage("", chanID, text, msg)
			return
		}
		s.IncomingMessage("", directUserName, text, msg)
	} else {
		s.IncomingMessage(ci.Name, userName, text, msg)
	}
}
