package util

import "testing"

func TestMatchEmojiPrefix(t *testing.T) {
	emoji, width := MatchEmojiPrefix("🚀 launch")
	if emoji != "🚀" || width != 2 {
		t.Fatalf("MatchEmojiPrefix() = %q, %d; want %q, %d", emoji, width, "🚀", 2)
	}

	emoji, width = MatchEmojiPrefix("plain text")
	if emoji != "" || width != 0 {
		t.Fatalf("MatchEmojiPrefix() = %q, %d; want empty match", emoji, width)
	}
}

func TestStringDisplayWidthUsesEmojiSequences(t *testing.T) {
	if got := StringDisplayWidth("go 🇦🇫 now"); got != 9 {
		t.Fatalf("StringDisplayWidth() = %d, want %d", got, 9)
	}
}

func TestStringDisplayWidthCountsTabsLikeReadline(t *testing.T) {
	if got := StringDisplayWidth("a\tb"); got != 6 {
		t.Fatalf("StringDisplayWidth() = %d, want %d", got, 6)
	}
}

func TestExpandTabsUsesLineLocalTabStops(t *testing.T) {
	in := "foo\nab\tcde\t\tfg"
	want := "foo\nab  cde     fg"
	if got := ExpandTabs(in, TerminalTabWidth); got != want {
		t.Fatalf("ExpandTabs() = %q, want %q", got, want)
	}
}

func TestExpandTabsIgnoresANSIWidth(t *testing.T) {
	in := "\x1b[31mab\tc"
	want := "\x1b[31mab  c"
	if got := ExpandTabs(in, TerminalTabWidth); got != want {
		t.Fatalf("ExpandTabs() = %q, want %q", got, want)
	}
}

func TestExpandTabsUsesEmojiDisplayWidth(t *testing.T) {
	in := "🚀\tx"
	want := "🚀  x"
	if got := ExpandTabs(in, TerminalTabWidth); got != want {
		t.Fatalf("ExpandTabs() = %q, want %q", got, want)
	}
}
