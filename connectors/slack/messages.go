package slack

/* util has most of the struct, type, and const definitions, as well as
most of the internal methods. */

import (
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
)

// Soft hyphen. *shrug*
const escapePad = "\u00AD"

type userlast struct {
	user, channel string
}

type slackOutgoingPayload struct {
	text       string
	legacyText string
	blocks     []slack.Block
}

// If we get back an edited message from a user in a channel within the
// ignorewindow ... well, we ignore it. The problem is, the Slack service will
// on occasion edit a user message, and the robot was seeing this as the user
// sending the same command twice in short order.
const ignorewindow = 3 * time.Second
const slackFormattedMaxSize = slack.MaxMessageTextLength - 490
const slackBlockTextLimit = 3000
const slackTruncatedMessage = "(message too long, truncated)"

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

func (s *slackConnector) applySlackPrefix(targetUserID, prefix, msg string, msgObject *robot.ConnectorMessage) string {
	mtype := getMsgType(msgObject)
	if len(prefix) > 0 && (mtype != msgSlashCmd || (targetUserID != msgObject.UserID)) {
		return prefix + msg
	}
	return msg
}

func (s *slackConnector) formatSlackMessage(msg string, f robot.MessageFormat) string {
	if f == robot.Raw {
		msg = normalizeBackticks(msg)
		msg = s.processRawMessage(msg)
	} else if f == robot.BasicMarkdown {
		msg = s.renderBasicMarkdown(msg)
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
	return msg
}

func (s *slackConnector) segmentFormattedSlackMessage(msg string) []string {
	msgLen := len(msg)
	if msgLen <= slackFormattedMaxSize {
		return []string{msg}
	}
	// It's too big, gotta chop it up. We will send at most maxMessageSplit
	// messages, plus "(message truncated)".
	msgs := make([]string, 0, s.maxMessageSplit+1)
	s.Log(robot.Info, "Message too long, segmenting: %d bytes", msgLen)
	// Chop it up into <=maxSize pieces
	var chunk string
	inside_block := false
	for len(msg) > slackFormattedMaxSize && len(msgs) < s.maxMessageSplit {
		lineEnd := strings.LastIndexByte(msg[:slackFormattedMaxSize], '\n')
		if lineEnd <= 0 { // no usable newline in this chunk
			chunk = msg[:slackFormattedMaxSize]
			msg = msg[slackFormattedMaxSize:]
		} else {
			chunk = msg[:lineEnd]
			msg = msg[lineEnd+1:] // skip over the newline
		}
		inside_block, chunk = optAddBlockDelimeters(inside_block, chunk)
		msgs = append(msgs, chunk)
	}
	if len(msgs) == s.maxMessageSplit { // we've maxed out
		if len(msg) > 0 { // if there's anything left, we've truncated
			msgs = append(msgs, slackTruncatedMessage)
		}
	} else { // the last chunk fits
		_, chunk = optAddBlockDelimeters(inside_block, msg)
		msgs = append(msgs, chunk)
	}
	return msgs
}

func splitSlackBlockText(msg string, maxChunkSize, maxMessageSplit int) []string {
	if len(msg) <= maxChunkSize {
		return []string{msg}
	}
	msgs := make([]string, 0, maxMessageSplit+1)
	for len(msg) > maxChunkSize && len(msgs) < maxMessageSplit {
		lineEnd := strings.LastIndexByte(msg[:maxChunkSize], '\n')
		var chunk string
		if lineEnd <= 0 {
			chunk = msg[:maxChunkSize]
			msg = msg[maxChunkSize:]
		} else {
			chunk = msg[:lineEnd]
			msg = msg[lineEnd+1:]
		}
		msgs = append(msgs, chunk)
	}
	if len(msgs) == maxMessageSplit {
		if len(msg) > 0 {
			msgs = append(msgs, slackTruncatedMessage)
		}
	} else {
		msgs = append(msgs, msg)
	}
	return msgs
}

func buildSlackVariableBlocks(text string) []slack.Block {
	return []slack.Block{
		slack.NewRichTextBlock("", slack.NewRichTextSection(
			slack.NewRichTextSectionTextElement(text, nil),
		)),
	}
}

func buildSlackFixedBlocks(text string) []slack.Block {
	preformatted := &slack.RichTextPreformatted{
		RichTextSection: slack.RichTextSection{
			Type: slack.RTEPreformatted,
			Elements: []slack.RichTextSectionElement{
				slack.NewRichTextSectionTextElement(text, nil),
			},
		},
	}
	return []slack.Block{
		slack.NewRichTextBlock("", preformatted),
	}
}

func buildSlackTruncationBlocks(text string) []slack.Block {
	return []slack.Block{
		slack.NewSectionBlock(slack.NewTextBlockObject(slack.PlainTextType, text, false, false), nil, nil),
	}
}

func (s *slackConnector) slackifyLegacyMessage(targetUserID, prefix, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) []string {
	msg = s.formatSlackMessage(msg, f)
	msg = s.applySlackPrefix(targetUserID, prefix, msg, msgObject)
	return s.segmentFormattedSlackMessage(msg)
}

// slackifyMessage replaces @username with the slack-internal representation, handles escaping,
// and returns either text-only or block-backed outbound payloads.
func (s *slackConnector) slackifyMessage(targetUserID, legacyPrefix, blockPrefix, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) []slackOutgoingPayload {
	if f != robot.Variable && f != robot.Fixed {
		msgs := s.slackifyLegacyMessage(targetUserID, legacyPrefix, msg, f, msgObject)
		payloads := make([]slackOutgoingPayload, 0, len(msgs))
		for _, formatted := range msgs {
			payloads = append(payloads, slackOutgoingPayload{
				text:       formatted,
				legacyText: formatted,
			})
		}
		return payloads
	}

	blockText := s.applySlackPrefix(targetUserID, blockPrefix, msg, msgObject)
	chunks := splitSlackBlockText(blockText, slackBlockTextLimit, s.maxMessageSplit)
	payloads := make([]slackOutgoingPayload, 0, len(chunks))
	for _, chunk := range chunks {
		payload := slackOutgoingPayload{
			text:       chunk,
			legacyText: s.formatSlackMessage(chunk, f),
		}
		switch {
		case chunk == slackTruncatedMessage:
			payload.blocks = buildSlackTruncationBlocks(chunk)
		case f == robot.Variable:
			payload.blocks = buildSlackVariableBlocks(chunk)
		default:
			payload.blocks = buildSlackFixedBlocks(chunk)
		}
		payloads = append(payloads, payload)
	}
	return payloads
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
