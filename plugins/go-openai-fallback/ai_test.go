package main

import (
	"regexp"
	"testing"
)

func TestNormalizeChunkTextHeadingsAndLists(t *testing.T) {
	in := "# Plan\n1. First\n2) Second\n+ Third\n\u2022 Fourth"
	want := "**Plan**\n- First\n- Second\n- Third\n- Fourth"

	got := normalizeChunkText(in)
	if got != want {
		t.Fatalf("normalizeChunkText() = %q, want %q", got, want)
	}
}

func TestNormalizeChunkTextLinksHTMLAndStrike(t *testing.T) {
	in := "See <https://example.com|docs> and <https://example.com>\n<b>bold</b> <em>ital</em> <code>x</code> ~~gone~~"
	want := "See [docs](https://example.com) and https://example.com\n**bold** *ital* `x` gone"

	got := normalizeChunkText(in)
	if got != want {
		t.Fatalf("normalizeChunkText() = %q, want %q", got, want)
	}
}

func TestNormalizeChunkTextPreservesCodeFenceContent(t *testing.T) {
	in := "```go\n<b>keep</b> __x__\n```\n# Header"
	want := "```go\n<b>keep</b> __x__\n```\n**Header**"

	got := normalizeChunkText(in)
	if got != want {
		t.Fatalf("normalizeChunkText() = %q, want %q", got, want)
	}
}

func TestNormalizeChunkTextUnderscoreWordNotItalic(t *testing.T) {
	in := "Use foo_bar as-is."
	got := normalizeChunkText(in)
	if got != in {
		t.Fatalf("normalizeChunkText() = %q, want %q", got, in)
	}
}

func TestHasUnbalancedFences(t *testing.T) {
	if !hasUnbalancedFences("```go\nfmt.Println(\"x\")") {
		t.Fatal("expected unbalanced fence to be detected")
	}
	if hasUnbalancedFences("```go\nfmt.Println(\"x\")\n```") {
		t.Fatal("did not expect balanced fence to be marked unbalanced")
	}
}

func TestConversationDatumKeyDeterministicAndSafe(t *testing.T) {
	key1 := conversationDatumKey("thread:slack:botdev:1700.33")
	key2 := conversationDatumKey("thread:slack:botdev:1700.33")
	key3 := conversationDatumKey("thread:slack:botdev:1700.34")
	if key1 != key2 {
		t.Fatalf("expected deterministic key generation, got %q vs %q", key1, key2)
	}
	if key1 == key3 {
		t.Fatalf("expected different conversation IDs to hash to different keys, got %q", key1)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_:]+$`).MatchString(key1) {
		t.Fatalf("conversation key contains unsupported characters: %q", key1)
	}
	if !regexp.MustCompile(`^` + regexp.QuoteMeta(conversationDatumPrefix) + `:[A-Fa-f0-9]{40}$`).MatchString(key1) {
		t.Fatalf("conversation key does not match expected prefix/hash format: %q", key1)
	}
}

func TestConversationIndexEntryHelpers(t *testing.T) {
	idx := conversationIndex{}
	upsertConversationIndexEntry(&idx, "thread:slack:botdev:1", "openaifallback:conversation:v2:abc", "2026-03-02T12:00:00Z")
	if idx.Version != conversationIndexVersion {
		t.Fatalf("index version = %d, want %d", idx.Version, conversationIndexVersion)
	}
	if len(idx.Conversations) != 1 {
		t.Fatalf("index size = %d, want 1", len(idx.Conversations))
	}
	entry, ok := idx.Conversations["thread:slack:botdev:1"]
	if !ok {
		t.Fatal("expected conversation entry to be present")
	}
	if entry.Key != "openaifallback:conversation:v2:abc" {
		t.Fatalf("entry key = %q, want %q", entry.Key, "openaifallback:conversation:v2:abc")
	}
	deleteConversationIndexEntry(&idx, "thread:slack:botdev:1")
	if len(idx.Conversations) != 0 {
		t.Fatalf("index size after delete = %d, want 0", len(idx.Conversations))
	}
}
