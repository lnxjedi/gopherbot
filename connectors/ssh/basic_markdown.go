package ssh

import "strings"

type basicMarkdownEmphasisMode int

const (
	basicMarkdownEmphasisPlain basicMarkdownEmphasisMode = iota
	basicMarkdownEmphasisANSI
)

var basicMarkdownCoreEmoji = map[string]string{
	"white_check_mark": "\u2705",
	"warning":          "\u26a0\ufe0f",
	"x":                "\u274c",
	"rocket":           "\U0001f680",
	"fire":             "\U0001f525",
	"joy":              "\U0001f602",
	"thinking_face":    "\U0001f914",
	"eyes":             "\U0001f440",
	"thumbsup":         "\U0001f44d",
	"thumbsdown":       "\U0001f44e",
}

func renderBasicMarkdownPlain(msg string) string {
	return renderBasicMarkdown(msg, basicMarkdownEmphasisPlain)
}

func renderBasicMarkdownStyled(msg string) string {
	return renderBasicMarkdown(msg, basicMarkdownEmphasisANSI)
}

func renderBasicMarkdown(msg string, emphasisMode basicMarkdownEmphasisMode) string {
	var out strings.Builder
	inFence := false

	for {
		idx := strings.Index(msg, "```")
		if idx == -1 {
			if inFence {
				out.WriteString(msg)
			} else {
				out.WriteString(renderBasicMarkdownInline(msg, emphasisMode))
			}
			break
		}

		chunk := msg[:idx]
		if inFence {
			out.WriteString(chunk)
		} else {
			out.WriteString(renderBasicMarkdownInline(chunk, emphasisMode))
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

func renderBasicMarkdownInline(msg string, emphasisMode basicMarkdownEmphasisMode) string {
	var out strings.Builder
	for len(msg) > 0 {
		start := findNextUnescapedBacktick(msg, 0)
		if start == -1 {
			out.WriteString(renderBasicMarkdownChunk(msg, emphasisMode))
			break
		}
		out.WriteString(renderBasicMarkdownChunk(msg[:start], emphasisMode))

		end := findNextUnescapedBacktick(msg, start+1)
		if end == -1 {
			out.WriteString(renderBasicMarkdownChunk(msg[start:], emphasisMode))
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

func renderBasicMarkdownChunk(msg string, emphasisMode basicMarkdownEmphasisMode) string {
	msg, escapedLiterals := protectBasicMarkdownEscapes(msg)
	msg = replaceBasicMarkdownLinks(msg)
	msg = replaceBasicMarkdownEmoji(msg)
	msg = renderBasicMarkdownEmphasis(msg, emphasisMode)
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
		if emoji, ok := basicMarkdownCoreEmoji[name]; ok {
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

func renderBasicMarkdownEmphasis(msg string, mode basicMarkdownEmphasisMode) string {
	msg = renderBasicMarkdownBold(msg, mode)
	msg = renderBasicMarkdownItalic(msg, mode)
	return msg
}

func renderBasicMarkdownBold(msg string, mode basicMarkdownEmphasisMode) string {
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

		out.WriteString(renderBasicMarkdownEmphasizedText(msg[:end], mode, "\x1b[1m", "\x1b[22m"))
		msg = msg[end+2:]
	}

	return out.String()
}

func renderBasicMarkdownItalic(msg string, mode basicMarkdownEmphasisMode) string {
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

		out.WriteString(renderBasicMarkdownEmphasizedText(msg[i+1:end], mode, "\x1b[3m", "\x1b[23m"))
		i = end + 1
	}

	return out.String()
}

func renderBasicMarkdownEmphasizedText(msg string, mode basicMarkdownEmphasisMode, startANSI, endANSI string) string {
	if mode != basicMarkdownEmphasisANSI || msg == "" {
		return msg
	}
	return startANSI + msg + endANSI
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
	return "\x00GBESC" + strconvItoa(idx) + "\x00"
}

func restoreEscapedLiterals(msg string, literals []string) string {
	out := msg
	for i, literal := range literals {
		out = strings.ReplaceAll(out, escapedPlaceholder(i), literal)
	}
	return out
}

func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
