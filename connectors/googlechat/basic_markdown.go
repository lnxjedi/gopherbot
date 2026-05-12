package googlechat

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/robot/util"
)

// Emoji rendering now uses the comprehensive emoji map from util.EmojiUnicode
// with automatic shortcode normalization (dashes converted to underscores).

const googleChatZWSP = "\u200B"

const (
	googleChatHomoglyphAsterisk    = "\u2217"
	googleChatHomoglyphUnderscore  = "\uff3f"
	googleChatHomoglyphTilde       = "\uff5e"
	googleChatHomoglyphBacktick    = "\uff40"
	googleChatHomoglyphLessThan    = "\uff1c"
	googleChatHomoglyphGreaterThan = "\uff1e"
	googleChatHomoglyphHyphen      = "\u2010"
)

func (gc *googleChatConnector) renderMessageText(msg string, format robot.MessageFormat) string {
	switch format {
	case robot.BasicMarkdown:
		return gc.renderBasicMarkdown(msg)
	case robot.Fixed:
		return gc.renderFixed(msg)
	case robot.Variable:
		return renderGoogleChatVariableLiteralText(msg)
	case robot.Raw:
		return msg
	default:
		return msg
	}
}

func (gc *googleChatConnector) renderFixed(msg string) string {
	if strings.TrimSpace(msg) == "" {
		return ""
	}
	return "```\n" + renderGoogleChatCodeLiteralText(msg) + "\n```"
}

func (gc *googleChatConnector) renderBasicMarkdown(msg string) string {
	var out strings.Builder
	inFence := false

	for {
		idx := strings.Index(msg, "```")
		if idx == -1 {
			if inFence {
				out.WriteString(renderGoogleChatCodeLiteralText(msg))
			} else {
				out.WriteString(gc.renderBasicMarkdownInline(msg))
			}
			break
		}

		chunk := msg[:idx]
		if inFence {
			out.WriteString(renderGoogleChatCodeLiteralText(chunk))
		} else {
			out.WriteString(gc.renderBasicMarkdownInline(chunk))
		}
		out.WriteString("```")
		inFence = !inFence
		msg = msg[idx+3:]
		if inFence {
			msg = stripBasicMarkdownFenceLanguage(msg)
		}
	}

	return out.String()
}

func stripBasicMarkdownFenceLanguage(msg string) string {
	if msg == "" || msg[0] == '\n' {
		return msg
	}
	lineEnd := strings.IndexByte(msg, '\n')
	if lineEnd == -1 {
		return ""
	}
	return msg[lineEnd:]
}

func (gc *googleChatConnector) renderBasicMarkdownInline(msg string) string {
	var out strings.Builder
	for len(msg) > 0 {
		start := findNextUnescapedBacktick(msg, 0)
		if start == -1 {
			out.WriteString(gc.renderBasicMarkdownChunk(msg))
			break
		}
		out.WriteString(gc.renderBasicMarkdownChunk(msg[:start]))

		end := findNextUnescapedBacktick(msg, start+1)
		if end == -1 {
			out.WriteString(gc.renderBasicMarkdownChunk(msg[start:]))
			break
		}
		out.WriteByte('`')
		out.WriteString(renderGoogleChatCodeLiteralText(msg[start+1 : end]))
		out.WriteByte('`')
		msg = msg[end+1:]
	}
	return out.String()
}

func findNextUnescapedBacktick(msg string, start int) int {
	for i := start; i < len(msg); i++ {
		if msg[i] == '`' && !isEscapedAt(msg, i) {
			return i
		}
	}
	return -1
}

func isEscapedAt(msg string, idx int) bool {
	if idx <= 0 || idx > len(msg)-1 {
		return false
	}
	slashes := 0
	for i := idx - 1; i >= 0 && msg[i] == '\\'; i-- {
		slashes++
	}
	return slashes%2 == 1
}

func (gc *googleChatConnector) renderBasicMarkdownChunk(msg string) string {
	msg, escaped := protectBasicMarkdownEscapes(msg)
	msg = replaceBasicMarkdownLinks(msg)
	msg = gc.replaceBasicMarkdownMentions(msg)
	msg = replaceBasicMarkdownEmoji(msg)
	mdTokens := make([]string, 0)
	reserveMD := func(token string) string {
		mdTokens = append(mdTokens, token)
		return markdownPlaceholder(len(mdTokens) - 1)
	}
	msg = replaceBasicMarkdownBold(msg, reserveMD)
	msg = replaceBasicMarkdownItalic(msg)
	msg = restoreMarkdownPlaceholders(msg, mdTokens)
	msg = restoreEscapedLiterals(msg, escaped)
	return msg
}

func protectBasicMarkdownEscapes(msg string) (string, []string) {
	escaped := make([]string, 0)
	var out strings.Builder
	for i := 0; i < len(msg); i++ {
		ch := msg[i]
		if ch != '\\' || i+1 >= len(msg) || !isBasicMarkdownEscapable(msg[i+1]) {
			out.WriteByte(ch)
			continue
		}
		escaped = append(escaped, string(msg[i+1]))
		out.WriteString(escapedPlaceholder(len(escaped) - 1))
		i++
	}
	return out.String(), escaped
}

func isBasicMarkdownEscapable(ch byte) bool {
	switch ch {
	case '*', '`', '[', ']', '(', ')', '@', '\\':
		return true
	default:
		return false
	}
}

func escapedPlaceholder(idx int) string {
	return fmt.Sprintf("\x00GBESC%d\x00", idx)
}

func markdownPlaceholder(idx int) string {
	return fmt.Sprintf("\x00GBMD%d\x00", idx)
}

func restoreEscapedLiterals(msg string, escaped []string) string {
	for idx, literal := range escaped {
		msg = strings.ReplaceAll(msg, escapedPlaceholder(idx), renderGoogleChatEscapedLiteral(literal))
	}
	return msg
}

func restoreMarkdownPlaceholders(msg string, mdTokens []string) string {
	for idx, token := range mdTokens {
		msg = strings.ReplaceAll(msg, markdownPlaceholder(idx), token)
	}
	return msg
}

func replaceBasicMarkdownLinks(msg string) string {
	var out strings.Builder
	for i := 0; i < len(msg); {
		open := strings.IndexByte(msg[i:], '[')
		if open == -1 {
			out.WriteString(msg[i:])
			break
		}
		open += i
		out.WriteString(msg[i:open])

		close := strings.IndexByte(msg[open+1:], ']')
		if close == -1 {
			out.WriteByte(msg[open])
			i = open + 1
			continue
		}
		close += open + 1
		if close+1 >= len(msg) || msg[close+1] != '(' {
			out.WriteString(msg[open : close+1])
			i = close + 1
			continue
		}
		end := strings.IndexByte(msg[close+2:], ')')
		if end == -1 {
			out.WriteString(msg[open:])
			break
		}
		end += close + 2

		label := renderGoogleChatLabelLiteralText(replaceBasicMarkdownEmoji(msg[open+1 : close]))
		url := msg[close+2 : end]
		if !isBasicMarkdownLinkURL(url) {
			out.WriteString(msg[open : end+1])
			i = end + 1
			continue
		}
		out.WriteString("<")
		out.WriteString(url)
		if strings.TrimSpace(label) != "" {
			out.WriteString("|")
			out.WriteString(label)
		}
		out.WriteString(">")
		i = end + 1
	}
	return out.String()
}

func isBasicMarkdownLinkURL(url string) bool {
	if strings.ContainsAny(url, " \t\r\n") {
		return false
	}
	return strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")
}

func renderGoogleChatVariableLiteralText(msg string) string {
	var out strings.Builder
	lineStart := true
	for i := 0; i < len(msg); i++ {
		ch := msg[i]
		if lineStart && i+1 < len(msg) && msg[i+1] == ' ' {
			switch ch {
			case '-':
				out.WriteString(googleChatHomoglyphHyphen)
				lineStart = false
				continue
			case '>':
				out.WriteString(googleChatHomoglyphGreaterThan)
				lineStart = false
				continue
			}
		}
		switch ch {
		case '*':
			out.WriteString(googleChatHomoglyphAsterisk)
		case '_':
			out.WriteString(googleChatHomoglyphUnderscore)
		case '~':
			out.WriteString(googleChatHomoglyphTilde)
		case '`':
			out.WriteString(googleChatHomoglyphBacktick)
		case '<':
			out.WriteString(googleChatHomoglyphLessThan)
		case '>':
			out.WriteString(googleChatHomoglyphGreaterThan)
		default:
			out.WriteByte(ch)
		}
		lineStart = ch == '\n'
	}
	return out.String()
}

func renderGoogleChatLabelLiteralText(msg string) string {
	var out strings.Builder
	for i := 0; i < len(msg); i++ {
		ch := msg[i]
		switch ch {
		case '*', '_', '~', '`', '<', '>', '|':
			out.WriteString(googleChatZWSP)
			out.WriteByte(ch)
			out.WriteString(googleChatZWSP)
		default:
			out.WriteByte(ch)
		}
	}
	return out.String()
}

func renderGoogleChatCodeLiteralText(msg string) string {
	var out strings.Builder
	lastWasZWSP := false
	inURL := false
	writeZWSP := func() {
		if lastWasZWSP {
			return
		}
		out.WriteString(googleChatZWSP)
		lastWasZWSP = true
	}
	writeByte := func(ch byte) {
		out.WriteByte(ch)
		lastWasZWSP = false
	}

	for i := 0; i < len(msg); i++ {
		ch := msg[i]
		switch ch {
		case '<', '>', '|':
			writeZWSP()
			writeByte(ch)
			writeZWSP()
			inURL = ch == '<'
			continue
		case ':':
			if i+2 < len(msg) && msg[i+1] == '/' && msg[i+2] == '/' && looksLikeURLSchemePrefix(msg, i) {
				writeZWSP()
				writeByte(ch)
				writeZWSP()
				inURL = true
				continue
			}
		case '/', '.', '?', '#', '&', '=', '%':
			if inURL {
				writeZWSP()
				writeByte(ch)
				writeZWSP()
				continue
			}
		}
		if inURL {
			switch ch {
			case ' ', '\t', '\r', '\n', '>', ')', ']', '}':
				inURL = false
			}
		}
		writeByte(ch)
	}
	return out.String()
}

func looksLikeURLSchemePrefix(msg string, colon int) bool {
	if colon <= 0 {
		return false
	}
	start := colon - 1
	for start >= 0 {
		ch := msg[start]
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '+' || ch == '-' || ch == '.':
		default:
			start++
			goto done
		}
		start--
	}
	start = 0
done:
	if start >= colon {
		return false
	}
	first := msg[start]
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')
}

func renderGoogleChatEscapedLiteral(literal string) string {
	switch literal {
	case "*", "_", "~", "`", "<", ">", "|":
		return googleChatZWSP + literal + googleChatZWSP
	default:
		return literal
	}
}

func replaceBasicMarkdownEmoji(msg string) string {
	var out strings.Builder
	for i := 0; i < len(msg); {
		if msg[i] != ':' {
			out.WriteByte(msg[i])
			i++
			continue
		}

		end := findBasicMarkdownEmojiEnd(msg, i)
		if end == -1 {
			out.WriteByte(msg[i])
			i++
			continue
		}

		name := msg[i+1 : end]
		if emoji := util.EmojiUnicode(name); emoji != "" {
			out.WriteString(emoji)
		} else {
			out.WriteString(msg[i : end+1])
		}
		i = end + 1
	}
	return out.String()
}

func findBasicMarkdownEmojiEnd(msg string, start int) int {
	if start < 0 || start >= len(msg) || msg[start] != ':' {
		return -1
	}
	if start > 0 && isBasicMarkdownEmojiNameChar(msg[start-1]) {
		return -1
	}
	nameStart := start + 1
	if nameStart >= len(msg) || !isBasicMarkdownEmojiNameChar(msg[nameStart]) {
		return -1
	}
	for i := nameStart; i < len(msg); i++ {
		switch {
		case msg[i] == ':':
			if i+1 < len(msg) && isBasicMarkdownEmojiNameChar(msg[i+1]) {
				return -1
			}
			return i
		case isBasicMarkdownEmojiNameChar(msg[i]):
			continue
		default:
			return -1
		}
	}
	return -1
}

func isBasicMarkdownEmojiNameChar(ch byte) bool {
	switch {
	case ch >= 'a' && ch <= 'z':
		return true
	case ch >= 'A' && ch <= 'Z':
		return true
	case ch >= '0' && ch <= '9':
		return true
	case ch == '_' || ch == '+' || ch == '-':
		return true
	default:
		return false
	}
}

func replaceBasicMarkdownBold(msg string, reserveMD func(string) string) string {
	var out strings.Builder
	for len(msg) > 0 {
		start := strings.Index(msg, "**")
		if start == -1 {
			out.WriteString(msg)
			break
		}
		out.WriteString(msg[:start])
		msg = msg[start+2:]

		end := strings.Index(msg, "**")
		if end == -1 {
			out.WriteString("**")
			out.WriteString(msg)
			break
		}
		inner := msg[:end]
		if inner == "" {
			out.WriteString("****")
		} else {
			out.WriteString(reserveMD("*" + inner + "*"))
		}
		msg = msg[end+2:]
	}
	return out.String()
}

func replaceBasicMarkdownItalic(msg string) string {
	var out strings.Builder
	for i := 0; i < len(msg); {
		if msg[i] != '*' || isAdjacentAsterisk(msg, i) {
			out.WriteByte(msg[i])
			i++
			continue
		}
		end := findNextSingleAsterisk(msg, i+1)
		if end == -1 {
			out.WriteByte(msg[i])
			i++
			continue
		}
		inner := msg[i+1 : end]
		if inner == "" {
			out.WriteString("**")
			i = end + 1
			continue
		}
		out.WriteByte('_')
		out.WriteString(inner)
		out.WriteByte('_')
		i = end + 1
	}
	return out.String()
}

func findNextSingleAsterisk(msg string, start int) int {
	for i := start; i < len(msg); i++ {
		if msg[i] == '*' && !isAdjacentAsterisk(msg, i) {
			return i
		}
	}
	return -1
}

func isAdjacentAsterisk(msg string, idx int) bool {
	return (idx > 0 && msg[idx-1] == '*') || (idx+1 < len(msg) && msg[idx+1] == '*')
}

func (gc *googleChatConnector) replaceBasicMarkdownMentions(msg string) string {
	var out strings.Builder
	for i := 0; i < len(msg); {
		if msg[i] != '@' || isEmailMention(msg, i) {
			out.WriteByte(msg[i])
			i++
			continue
		}
		end := findMentionEnd(msg, i+1)
		if end == i+1 {
			out.WriteByte(msg[i])
			i++
			continue
		}
		name := strings.ToLower(msg[i+1 : end])
		gc.mu.RLock()
		userID, ok := gc.botUserMap[name]
		gc.mu.RUnlock()
		if ok {
			out.WriteString("<")
			out.WriteString(userID)
			out.WriteString(">")
		} else {
			out.WriteString(msg[i:end])
		}
		i = end
	}
	return out.String()
}

func isEmailMention(msg string, at int) bool {
	if at <= 0 {
		return false
	}
	prev := msg[at-1]
	return (prev >= 'A' && prev <= 'Z') || (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || prev == '.' || prev == '_' || prev == '-'
}

func findMentionEnd(msg string, start int) int {
	i := start
	for i < len(msg) {
		ch := msg[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' || ch == '.' {
			i++
			continue
		}
		break
	}
	return i
}

func renderPlainForLogs(msg string) string {
	return util.RenderBasicMarkdownPlain(msg)
}
