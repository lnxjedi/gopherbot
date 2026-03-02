package slack

import (
	"fmt"
	"strings"
)

var slackPlainEscapeReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
)

func (s *slackConnector) renderBasicMarkdown(msg string) string {
	var out strings.Builder
	inFence := false

	for {
		idx := strings.Index(msg, "```")
		if idx == -1 {
			if inFence {
				out.WriteString(msg)
			} else {
				out.WriteString(s.renderBasicMarkdownInline(msg))
			}
			break
		}

		chunk := msg[:idx]
		if inFence {
			out.WriteString(chunk)
		} else {
			out.WriteString(s.renderBasicMarkdownInline(chunk))
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

func (s *slackConnector) renderBasicMarkdownInline(msg string) string {
	var out strings.Builder
	for len(msg) > 0 {
		start := findNextUnescapedBacktick(msg, 0)
		if start == -1 {
			out.WriteString(s.renderBasicMarkdownPlain(msg))
			break
		}
		out.WriteString(s.renderBasicMarkdownPlain(msg[:start]))

		end := findNextUnescapedBacktick(msg, start+1)
		if end == -1 {
			// Unterminated inline-code delimiter: treat as plain text.
			out.WriteString(s.renderBasicMarkdownPlain(msg[start:]))
			break
		}
		out.WriteString(msg[start : end+1])
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

func (s *slackConnector) renderBasicMarkdownPlain(msg string) string {
	msg, escapedLiterals := protectBasicMarkdownEscapes(msg)

	mdTokens := make([]string, 0)
	reserveMD := func(token string) string {
		mdTokens = append(mdTokens, token)
		return markdownPlaceholder(len(mdTokens) - 1)
	}

	msg = replaceBasicMarkdownLinks(msg, reserveMD)
	msg = s.replaceBasicMarkdownMentions(msg, reserveMD)
	msg = replaceBasicMarkdownEmphasis(msg, reserveMD)
	msg = slackPlainEscapeReplacer.Replace(msg)
	msg = restoreEscapedLiterals(msg, escapedLiterals)
	msg = restoreMarkdownPlaceholders(msg, mdTokens)

	return msg
}

func replaceBasicMarkdownEmphasis(msg string, reserveMD func(string) string) string {
	msg = replaceBasicMarkdownBold(msg, reserveMD)
	msg = replaceBasicMarkdownItalic(msg)
	return msg
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

func protectBasicMarkdownEscapes(msg string) (string, []string) {
	escapedLiterals := make([]string, 0)
	var out strings.Builder

	for i := 0; i < len(msg); i++ {
		ch := msg[i]
		if ch != '\\' || i+1 >= len(msg) || !isBasicMarkdownEscapable(msg[i+1]) {
			out.WriteByte(ch)
			continue
		}
		escapedLiterals = append(escapedLiterals, string(msg[i+1]))
		out.WriteString(escapedPlaceholder(len(escapedLiterals) - 1))
		i++
	}

	return out.String(), escapedLiterals
}

func isBasicMarkdownEscapable(ch byte) bool {
	switch ch {
	case '*', '`', '[', ']', '(', ')', '@', '\\':
		return true
	default:
		return false
	}
}

func replaceBasicMarkdownLinks(msg string, reserveMD func(string) string) string {
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

		label := msg[open+1 : close]
		url := msg[close+2 : end]
		if !isBasicMarkdownLinkURL(url) {
			out.WriteString(msg[open : end+1])
			i = end + 1
			continue
		}

		if token, ok := toSlackLinkToken(label, url); ok {
			out.WriteString(reserveMD(token))
		} else {
			// If Slack cannot safely preserve label syntax, degrade.
			out.WriteString(label)
			out.WriteString(" (")
			out.WriteString(url)
			out.WriteByte(')')
		}
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

func toSlackLinkToken(label, url string) (string, bool) {
	if strings.ContainsAny(url, "|>") || strings.Contains(label, "|") {
		return "", false
	}
	escapedLabel := slackPlainEscapeReplacer.Replace(label)
	return "<" + url + "|" + escapedLabel + ">", true
}

func (s *slackConnector) replaceBasicMarkdownMentions(msg string, reserveMD func(string) string) string {
	var out strings.Builder

	for i := 0; i < len(msg); {
		if msg[i] == '<' {
			// Preserve existing slack-ish bracketed segments.
			if end := strings.IndexByte(msg[i+1:], '>'); end != -1 {
				end += i + 1
				out.WriteString(msg[i : end+1])
				i = end + 1
				continue
			}
		}

		if msg[i] != '@' {
			out.WriteByte(msg[i])
			i++
			continue
		}

		start := i
		i++
		for i < len(msg) && isMentionTokenChar(msg[i]) {
			i++
		}
		if i == start+1 {
			out.WriteByte('@')
			continue
		}

		mentionToken := msg[start+1 : i]
		mention, suffix := splitMentionCandidate(mentionToken)
		if mention == "" {
			out.WriteString(msg[start:i])
			continue
		}
		if start > 0 && isEmailLocalChar(msg[start-1]) {
			out.WriteString(msg[start:i])
			continue
		}

		if userID, ok := s.resolveBasicMarkdownMentionID(mention); ok {
			out.WriteString(reserveMD("<@" + userID + ">"))
			out.WriteString(suffix)
			continue
		}
		out.WriteByte('@')
		out.WriteString(mention)
		out.WriteString(suffix)
	}

	return out.String()
}

func (s *slackConnector) resolveBasicMarkdownMentionID(user string) (string, bool) {
	if strings.TrimSpace(user) == "" {
		return "", false
	}

	s.RLock()
	defer s.RUnlock()
	if s.userMap == nil {
		return "", false
	}

	if id, ok := s.userMap[user]; ok && strings.TrimSpace(id) != "" {
		return id, true
	}

	found := ""
	for name, id := range s.userMap {
		if !strings.EqualFold(name, user) || strings.TrimSpace(id) == "" {
			continue
		}
		if found == "" {
			found = id
			continue
		}
		if found != id {
			return "", false
		}
	}
	if found == "" {
		return "", false
	}
	return found, true
}

func isMentionTokenChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '-' || ch == '.'
}

func splitMentionCandidate(token string) (mention string, suffix string) {
	if token == "" {
		return "", ""
	}
	cut := len(token)
	for cut > 0 && !isMentionTerminalChar(token[cut-1]) {
		cut--
	}
	return token[:cut], token[cut:]
}

func isMentionTerminalChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

func isEmailLocalChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '.' || ch == '%' || ch == '+' || ch == '-'
}

func escapedPlaceholder(idx int) string {
	return fmt.Sprintf("\x00GBESC%d\x00", idx)
}

func markdownPlaceholder(idx int) string {
	return fmt.Sprintf("\x00GBMD%d\x00", idx)
}

func restoreEscapedLiterals(msg string, literals []string) string {
	out := msg
	for i, literal := range literals {
		out = strings.ReplaceAll(out, escapedPlaceholder(i), slackEscapedLiteral(literal))
	}
	return out
}

func slackEscapedLiteral(literal string) string {
	switch literal {
	case "*", "`", "_":
		return escapePad + literal
	default:
		return literal
	}
}

func restoreMarkdownPlaceholders(msg string, tokens []string) string {
	out := msg
	for i, token := range tokens {
		out = strings.ReplaceAll(out, markdownPlaceholder(i), token)
	}
	return out
}
