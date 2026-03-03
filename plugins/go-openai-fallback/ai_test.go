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

func TestDeterministicCompactionPreservesRecentWindow(t *testing.T) {
	state := conversationState{
		Profile: defaultProfile,
		Exchanges: []conversationExchange{
			{Human: "alice says: one", AI: "ai: one"},
			{Human: "alice says: two", AI: "ai: two"},
			{Human: "alice says: three", AI: "ai: three"},
			{Human: "alice says: four", AI: "ai: four"},
			{Human: "alice says: five", AI: "ai: five"},
		},
	}
	state.Tokens = estimateConversationTokens(state.Exchanges)

	cfg := aiConfig{
		CompactionTriggerTokens: 1,
		MaxRecentExchanges:      2,
		SummaryBudgetTokens:     200,
	}
	compacted := maybeCompactConversationDeterministic(state, cfg)
	if len(compacted.Exchanges) != 2 {
		t.Fatalf("recent exchanges kept = %d, want 2", len(compacted.Exchanges))
	}
	if compacted.Exchanges[0].Human != "alice says: four" {
		t.Fatalf("unexpected first kept exchange: %q", compacted.Exchanges[0].Human)
	}
	if compacted.Exchanges[1].Human != "alice says: five" {
		t.Fatalf("unexpected second kept exchange: %q", compacted.Exchanges[1].Human)
	}
	if compacted.Summary == "" {
		t.Fatal("expected deterministic summary to be populated")
	}
}

func TestDeterministicCompactionNoopBelowTrigger(t *testing.T) {
	state := conversationState{
		Profile: defaultProfile,
		Exchanges: []conversationExchange{
			{Human: "alice says: one", AI: "ai: one"},
			{Human: "alice says: two", AI: "ai: two"},
			{Human: "alice says: three", AI: "ai: three"},
		},
	}
	state.Tokens = estimateConversationTokens(state.Exchanges)
	cfg := aiConfig{
		CompactionTriggerTokens: state.Tokens + 1000,
		MaxRecentExchanges:      2,
		SummaryBudgetTokens:     200,
	}
	compacted := maybeCompactConversationDeterministic(state, cfg)
	if len(compacted.Exchanges) != 3 {
		t.Fatalf("expected no compaction, exchanges = %d", len(compacted.Exchanges))
	}
	if compacted.Summary != "" {
		t.Fatalf("expected empty summary on no-op compaction, got %q", compacted.Summary)
	}
}

func TestBuildMessagesIncludesSummaryAfterSystem(t *testing.T) {
	messages := buildMessages(
		"system prompt",
		"older summary",
		[]conversationExchange{{Human: "alice says: hi", AI: "hello"}},
		nil,
		conversationContext{User: "alice", Prompt: "next"},
	)
	if len(messages) < 4 {
		t.Fatalf("messages len = %d, expected at least 4", len(messages))
	}
	if messages[0]["role"] != "system" || messages[0]["content"] != "system prompt" {
		t.Fatalf("unexpected first message: %#v", messages[0])
	}
	if messages[1]["role"] != "system" {
		t.Fatalf("expected summary as system message, got %#v", messages[1])
	}
	if !regexp.MustCompile(`Conversation summary`).MatchString(messages[1]["content"]) {
		t.Fatalf("expected summary marker, got %q", messages[1]["content"])
	}
}
