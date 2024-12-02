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

// normalizeBackticks adds leading and trailing newlines to triple backticks (```)
// If the leading or trailing newline is already present, it doesn't duplicate it.
func normalizeBackticks(input string) string {
	// Regular expression that matches a triple backtick with any character (or none) on either side
	re := regexp.MustCompile(".{0,1}```.{0,1}")

	// Replace function adds newlines before and after every triple backtick
	// If the newline is already present, it doesn't duplicate it
	return re.ReplaceAllStringFunc(input, func(s string) string {
		if s == "\n```\n" {
			return s
		}
		if strings.Count(s, "`") != 3 {
			return s
		}
		leading := string(s[0])
		trailing := string(s[len(s)-1])
		if leading == "`" {
			leading = ""
		} else if leading != "\n" {
			leading = leading + "\n"
		}
		if trailing == "`" {
			trailing = ""
		} else if trailing != "\n" {
			trailing = "\n" + trailing
		}
		return leading + "```" + trailing
	})
}

func optAddBlockDelimeters(inside_block bool, chunk string) (bool, string) {
	delimeters := strings.Count(chunk, "```")
	if inside_block {
		if strings.HasPrefix(chunk, "\n") {
			chunk = "```" + chunk
		} else {
			chunk = "```\n" + chunk
		}
	}
	if delimeters%2 == 1 {
		inside_block = !inside_block
	}
	if inside_block {
		if strings.HasSuffix(chunk, "\n") {
			chunk = chunk + "```"
		} else {
			chunk = chunk + "\n```"
		}
	}
	return inside_block, chunk
}

func (s *slackConnector) processRawMessage(msg string) string {
	var result strings.Builder
	inside_block := false
	chunks := strings.Split(msg, "```")
	num_chunks := len(chunks)

	for i, chunk := range chunks {
		if !inside_block {
			chunk = s.replaceMentions(chunk)
		}
		result.WriteString(chunk)
		// If there are more chunks, write the delimeter and flip the bool
		if i != num_chunks-1 {
			result.WriteString("```")
			inside_block = !inside_block
		}
	}

	if inside_block {
		result.WriteString("\n```")
	}

	return result.String()
}

// slackifyMessage replaces @username with the slack-internal representation, handles escaping,
// takes care of formatting, and segments the message if needed.
func (s *slackConnector) slackifyMessage(targetUserID, prefix, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) []string {
	maxSize := slack.MaxMessageTextLength - 490

	if f == robot.Raw {
		msg = normalizeBackticks(msg)
		msg = s.processRawMessage(msg)
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
	if len(prefix) > 0 && (mtype != msgSlashCmd || (targetUserID != msgObject.UserID)) {
		msg = prefix + msg
	}
	if f == robot.Fixed {
		f_prefix := "```\n"
		if strings.HasPrefix(msg, "\n") {
			f_prefix = "```"
		}
		f_suffix := "\n```"
		if strings.HasSuffix(msg, "\n") {
			f_suffix = "```"
		}
		msg = f_prefix + msg + f_suffix
	}

	msgLen := len(msg)
	if msgLen <= maxSize {
		return []string{msg}
	}
	// It's too big, gotta chop it up. We will send at most maxMessageSplit
	// messages, plus "(message truncated)".
	msgs := make([]string, 0, s.maxMessageSplit+1)
	s.Log(robot.Info, "Message too long, segmenting: %d bytes", msgLen)
	// Chop it up into <=maxSize pieces
	var chunk string
	inside_block := false
	for len(msg) > maxSize && len(msgs) < s.maxMessageSplit {
		lineEnd := strings.LastIndexByte(msg[:maxSize], '\n')
		if lineEnd == -1 { // no newline in this chunk
			chunk = msg[:maxSize]
			msg = msg[maxSize:]
		} else {
			chunk = msg[:lineEnd]
			msg = msg[lineEnd+1:] // skip over the newline
		}
		inside_block, chunk = optAddBlockDelimeters(inside_block, chunk)
		msgs = append(msgs, chunk)
	}
	if len(msgs) == s.maxMessageSplit { // we've maxed out
		if len(msg) > 0 { // if there's anything left, we've truncated
			msgs = append(msgs, "(message too long, truncated)")
		}
	} else { // the last chunk fits
		_, chunk = optAddBlockDelimeters(inside_block, msg)
		msgs = append(msgs, chunk)
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
