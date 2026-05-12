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