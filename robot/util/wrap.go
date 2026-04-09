package util

import (
	"strings"
	"unicode/utf8"
)

const (
	defaultBreakpoints = " -"
	defaultNewline     = "\n"
)

// Wrapper contains settings for customisable word-wrapping.
type Wrapper struct {
	Breakpoints string
	Newline     string

	OutputLinePrefix string
	OutputLineSuffix string

	LimitIncludesPrefixSuffix bool

	TrimInputPrefix string
	TrimInputSuffix string

	StripTrailingNewline bool
}

// NewWrapper returns a new instance of a Wrapper initialised with defaults.
func NewWrapper() Wrapper {
	return Wrapper{
		Breakpoints:               defaultBreakpoints,
		Newline:                   defaultNewline,
		LimitIncludesPrefixSuffix: true,
	}
}

// Wrap is shorthand for declaring a new default Wrapper calling its Wrap method.
func Wrap(s string, limit int) string {
	return NewWrapper().Wrap(s, limit)
}

// Wrap will wrap one or more lines of text at the given length.
func (w Wrapper) Wrap(s string, limit int) string {
	if w.LimitIncludesPrefixSuffix {
		limit -= visibleRuneCount(w.OutputLinePrefix) + visibleRuneCount(w.OutputLineSuffix)
	}

	var ret string
	for _, str := range strings.Split(s, w.Newline) {
		str = strings.TrimPrefix(str, w.TrimInputPrefix)
		str = strings.TrimSuffix(str, w.TrimInputSuffix)
		ret += w.line(str, limit) + w.Newline
	}

	if w.StripTrailingNewline {
		return strings.TrimSuffix(ret, w.Newline)
	}
	return ret
}

func (w Wrapper) line(s string, limit int) string {
	tokens := tokenizeWrapLine(s, w.Breakpoints)
	if limit < 1 || visibleTokenCount(tokens) < limit+1 {
		return w.OutputLinePrefix + s + w.OutputLineSuffix
	}

	i := lastVisibleBreakpoint(tokens, limit+1)
	if i < 0 {
		i = firstVisibleBreakpoint(tokens)
		if i < 0 {
			return w.OutputLinePrefix + s + w.OutputLineSuffix
		}
	}

	return w.OutputLinePrefix + joinWrapTokens(tokens[:i]) + w.OutputLineSuffix + w.Newline + w.line(joinWrapTokens(tokens[i+1:]), limit)
}

type wrapToken struct {
	text       string
	visible    bool
	breakpoint bool
}

func tokenizeWrapLine(s, breakpoints string) []wrapToken {
	tokens := make([]wrapToken, 0, len(s))
	for i := 0; i < len(s); {
		if seq, n := readANSISequence(s[i:]); n > 0 {
			tokens = append(tokens, wrapToken{text: seq})
			i += n
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if size <= 0 {
			break
		}
		tokens = append(tokens, wrapToken{
			text:       s[i : i+size],
			visible:    true,
			breakpoint: strings.ContainsRune(breakpoints, r),
		})
		i += size
	}
	return tokens
}

func readANSISequence(s string) (string, int) {
	if len(s) < 2 || s[0] != '\x1b' || s[1] != '[' {
		return "", 0
	}
	for i := 2; i < len(s); i++ {
		if s[i] >= 0x40 && s[i] <= 0x7e {
			return s[:i+1], i + 1
		}
	}
	return "", 0
}

func visibleRuneCount(s string) int {
	return visibleTokenCount(tokenizeWrapLine(s, defaultBreakpoints))
}

func visibleTokenCount(tokens []wrapToken) int {
	count := 0
	for _, token := range tokens {
		if token.visible {
			count++
		}
	}
	return count
}

func lastVisibleBreakpoint(tokens []wrapToken, limit int) int {
	visible := 0
	last := -1
	for i, token := range tokens {
		if token.visible {
			visible++
		}
		if token.breakpoint && visible <= limit {
			last = i
		}
	}
	return last
}

func firstVisibleBreakpoint(tokens []wrapToken) int {
	for i, token := range tokens {
		if token.breakpoint {
			return i
		}
	}
	return -1
}

func joinWrapTokens(tokens []wrapToken) string {
	var b strings.Builder
	for _, token := range tokens {
		b.WriteString(token.text)
	}
	return b.String()
}
