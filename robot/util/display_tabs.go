package util

import (
	"strings"
	"unicode/utf8"
)

const TerminalTabWidth = 4

// ExpandTabs replaces tabs with spaces at fixed terminal tab stops.
// ANSI escape sequences are copied without contributing to display width.
func ExpandTabs(s string, tabWidth int) string {
	if !strings.ContainsRune(s, '\t') {
		return s
	}
	if tabWidth <= 0 {
		tabWidth = TerminalTabWidth
	}
	var b strings.Builder
	b.Grow(len(s))
	col := 0
	for len(s) > 0 {
		if seq, n := readANSISequence(s); n > 0 {
			b.WriteString(seq)
			s = s[n:]
			continue
		}
		if emoji, emojiWidth := MatchEmojiPrefix(s); emojiWidth > 0 {
			b.WriteString(emoji)
			col += emojiWidth
			s = s[len(emoji):]
			continue
		}
		r, size := utf8.DecodeRuneInString(s)
		if size <= 0 {
			break
		}
		switch r {
		case '\t':
			spaces := tabWidth - (col % tabWidth)
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		case '\n', '\r':
			b.WriteRune(r)
			col = 0
		default:
			b.WriteString(s[:size])
			col += RuneDisplayWidth(r)
		}
		s = s[size:]
	}
	return b.String()
}
