package bot

import (
	"fmt"
	"strings"
)

func renderBasicMarkdownPlain(msg string) string {
	var out strings.Builder
	inFence := false

	for {
		idx := strings.Index(msg, "```")
		if idx == -1 {
			if inFence {
				out.WriteString(msg)
			} else {
				out.WriteString(renderBasicMarkdownInline(msg))
			}
			break
		}

		chunk := msg[:idx]
		if inFence {
			out.WriteString(chunk)
		} else {
			out.WriteString(renderBasicMarkdownInline(chunk))
		}

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

func renderBasicMarkdownInline(msg string) string {
	var out strings.Builder
	for len(msg) > 0 {
		start := findNextUnescapedBacktick(msg, 0)
		if start == -1 {
			out.WriteString(renderBasicMarkdownPlainChunk(msg))
			break
		}
		out.WriteString(renderBasicMarkdownPlainChunk(msg[:start]))

		end := findNextUnescapedBacktick(msg, start+1)
		if end == -1 {
			out.WriteString(renderBasicMarkdownPlainChunk(msg[start:]))
			break
		}
		out.WriteString(msg[start+1 : end])
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

func renderBasicMarkdownPlainChunk(msg string) string {
	msg, escapedLiterals := protectBasicMarkdownEscapes(msg)
	msg = replaceBasicMarkdownLinks(msg)
	msg = removeBasicMarkdownEmphasis(msg)
	msg = restoreEscapedLiterals(msg, escapedLiterals)
	return msg
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

		label := msg[open+1 : close]
		url := msg[close+2 : end]
		if !isBasicMarkdownLinkURL(url) {
			out.WriteString(msg[open : end+1])
			i = end + 1
			continue
		}

		out.WriteString(label)
		out.WriteString(" (")
		out.WriteString(url)
		out.WriteByte(')')
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

func removeBasicMarkdownEmphasis(msg string) string {
	msg = removeBasicMarkdownBold(msg)
	msg = removeBasicMarkdownItalic(msg)
	return msg
}

func removeBasicMarkdownBold(msg string) string {
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

		out.WriteString(msg[:end])
		msg = msg[end+2:]
	}

	return out.String()
}

func removeBasicMarkdownItalic(msg string) string {
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

		out.WriteString(msg[i+1 : end])
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

func escapedPlaceholder(idx int) string {
	return fmt.Sprintf("\x00GBESC%d\x00", idx)
}

func restoreEscapedLiterals(msg string, literals []string) string {
	out := msg
	for i, literal := range literals {
		out = strings.ReplaceAll(out, escapedPlaceholder(i), literal)
	}
	return out
}
