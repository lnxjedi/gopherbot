package slack

/* util has most of the struct, type, and const definitions, as well as
most of the internal methods. */

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
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
const ignorewindow = 3 * time.Second

func optQuote(msg string, f robot.MessageFormat) string {
	if f == robot.Fixed {
		return "```" + msg + "```"
	}
	return msg
}

var mentionMatch = `[0-9A-Za-z](?:[-_0-9A-Za-z.]{0,19}[_0-9A-Za-z])?`
var mentionRe = regexp.MustCompile(`@` + mentionMatch + `\b`)
var usernameRe = regexp.MustCompile(`^` + mentionMatch + `$`)

func (s *slackConnector) replaceMentions(msg string) string {
	return mentionRe.ReplaceAllStringFunc(msg, func(mentioned string) string {
		mentioned = mentioned[1:]
		switch mentioned {
		case "here", "channel", "everyone":
			return "<!" + mentioned + ">"
		}
		replace, ok := s.userID(mentioned, true)
		if ok {
			return "<@" + replace + ">"
		}
		return mentioned
	})
}

// slackifyMessage replaces @username with the slack-internal representation, handles escaping,
// takes care of formatting, and segments the message if needed.
func (s *slackConnector) slackifyMessage(prefix, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) []string {
	maxSize := slack.MaxMessageTextLength - 500 // workaround for large message disconnects
	if f == robot.Fixed {
		maxSize -= 6
	}

	if f == robot.Raw {
		msg = s.replaceMentions(msg)
	} else {
		msg = strings.Replace(msg, "&", "&amp;", -1)
		msg = strings.Replace(msg, "<", "&lt;", -1)
		msg = strings.Replace(msg, ">", "&gt;", -1)
	}
	if f == robot.Variable {
		// 'escape' special chars that aren't covered by disabling markdown.
		for _, padChar := range []string{"`", "*", "_", ":"} {
			paddedString := escapePad + padChar
			msg = strings.Replace(msg, padChar, paddedString, -1)
		}
	}
	mtype := getMsgType(msgObject)
	if len(prefix) > 0 && mtype != msgSlashCmd {
		msg = prefix + msg
	}

	msgLen := len(msg)
	if msgLen <= maxSize {
		return []string{optQuote(msg, f)}
	}
	// It's too big, gotta chop it up. We will send at most maxMessageSplit
	// messages, plus "(message truncated)".
	msgs := make([]string, 0, s.maxMessageSplit+1)
	s.Log(robot.Info, "Message too long, segmenting: %d bytes", msgLen)
	// Chop it up into <=maxSize pieces
	for len(msg) > maxSize && len(msgs) < s.maxMessageSplit {
		lineEnd := strings.LastIndexByte(msg[:maxSize], '\n')
		if lineEnd == -1 { // no newline in this chunk
			msgs = append(msgs, optQuote(msg[:maxSize], f))
			msg = msg[maxSize:]
		} else {
			msgs = append(msgs, optQuote(msg[:lineEnd], f))
			msg = msg[lineEnd+1:] // skip over the newline
		}
	}
	if len(msgs) == s.maxMessageSplit { // we've maxed out
		if len(msg) > 0 { // if there's anything left, we've truncated
			msgs = append(msgs, "(message too long, truncated)")
		}
	} else { // the last chunk fits
		msgs = append(msgs, optQuote(msg, f))
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

func validSubtype(st string) bool {
	switch st {
	case "bot_message", "message_replied", "file_share":
		return true
	default:
		return false
	}
}
