package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
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

func TestNormalizeChatCompletionPayloadConvertsMaxTokens(t *testing.T) {
	payload := map[string]interface{}{
		"model":      "gpt-5.2-chat-latest",
		"max_tokens": 321,
	}

	normalizeChatCompletionPayload(payload)

	if _, ok := payload["max_tokens"]; ok {
		t.Fatalf("expected deprecated max_tokens to be removed, payload=%v", payload)
	}
	if got := payload["max_completion_tokens"]; got != 321 {
		t.Fatalf("max_completion_tokens = %v, want 321", got)
	}
}

func TestNormalizeChatCompletionPayloadPrefersExistingMaxCompletionTokens(t *testing.T) {
	payload := map[string]interface{}{
		"model":                 "gpt-5.2-chat-latest",
		"max_tokens":            321,
		"max_completion_tokens": 654,
	}

	normalizeChatCompletionPayload(payload)

	if _, ok := payload["max_tokens"]; ok {
		t.Fatalf("expected deprecated max_tokens to be removed, payload=%v", payload)
	}
	if got := payload["max_completion_tokens"]; got != 654 {
		t.Fatalf("max_completion_tokens = %v, want 654", got)
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
	upsertConversationIndexEntry(&idx, "thread:slack:botdev:1", "aifallback:conversation:v2:abc", "2026-03-02T12:00:00Z")
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
	if entry.Key != "aifallback:conversation:v2:abc" {
		t.Fatalf("entry key = %q, want %q", entry.Key, "aifallback:conversation:v2:abc")
	}
	deleteConversationIndexEntry(&idx, "thread:slack:botdev:1")
	if len(idx.Conversations) != 0 {
		t.Fatalf("index size after delete = %d, want 0", len(idx.Conversations))
	}
}

func TestDecodeConversationStateFromRawState(t *testing.T) {
	source := conversationState{
		Profile: defaultProfile,
		Tokens:  42,
		Owner:   "alice",
		Summary: "summary",
		Exchanges: []conversationExchange{
			{Human: "alice says: hi", AI: "hello"},
		},
	}
	blob, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("marshal source state: %v", err)
	}
	var raw interface{}
	if err := json.Unmarshal(blob, &raw); err != nil {
		t.Fatalf("unmarshal raw state: %v", err)
	}

	got, ok := decodeConversationStateFromRaw(raw)
	if !ok {
		t.Fatal("expected decodeConversationStateFromRaw to decode state-shaped payload")
	}
	if got.Profile != source.Profile || got.Owner != source.Owner || got.Summary != source.Summary {
		t.Fatalf("decoded state metadata mismatch: got=%+v source=%+v", got, source)
	}
	if len(got.Exchanges) != 1 || got.Exchanges[0].Human != "alice says: hi" || got.Exchanges[0].AI != "hello" {
		t.Fatalf("decoded exchanges mismatch: %+v", got.Exchanges)
	}
}

func TestDecodeConversationStateFromRawRejectsExchangesArray(t *testing.T) {
	exchanges := []conversationExchange{
		{Human: "alice says: one", AI: "ai: one"},
		{Human: "alice says: two", AI: "ai: two"},
	}
	blob, err := json.Marshal(exchanges)
	if err != nil {
		t.Fatalf("marshal exchanges: %v", err)
	}
	var raw interface{}
	if err := json.Unmarshal(blob, &raw); err != nil {
		t.Fatalf("unmarshal raw exchanges: %v", err)
	}

	if _, ok := decodeConversationStateFromRaw(raw); ok {
		t.Fatal("expected decodeConversationStateFromRaw to reject exchange-only payload")
	}
}

func TestDecodeConversationStateFromRawBytesState(t *testing.T) {
	source := conversationState{
		Profile: defaultProfile,
		Owner:   "alice",
		Exchanges: []conversationExchange{
			{Human: "alice says: hi", AI: "hello"},
		},
	}
	blob, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("marshal source: %v", err)
	}

	got, ok := decodeConversationStateFromRaw(blob)
	if !ok {
		t.Fatal("expected decodeConversationStateFromRaw to decode []byte state payload")
	}
	if got.Owner != "alice" || len(got.Exchanges) != 1 {
		t.Fatalf("decoded state mismatch: %+v", got)
	}
}

func TestDecodeConversationStateFromRawBytesRejectsExchanges(t *testing.T) {
	exchanges := []conversationExchange{
		{Human: "alice says: one", AI: "ai: one"},
		{Human: "alice says: two", AI: "ai: two"},
	}
	blob, err := json.Marshal(exchanges)
	if err != nil {
		t.Fatalf("marshal exchanges: %v", err)
	}

	if _, ok := decodeConversationStateFromRaw(blob); ok {
		t.Fatal("expected decodeConversationStateFromRaw to reject []byte exchanges payload")
	}
}

func TestDecodeConversationIndexFromRawState(t *testing.T) {
	source := conversationIndex{
		Version: conversationIndexVersion,
		Conversations: map[string]conversationIndexEntry{
			"thread:slack:general:1": {
				Key:       "aifallback:conversation:v2:abc",
				UpdatedAt: "2026-03-04T12:00:00Z",
			},
		},
	}
	blob, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("marshal index: %v", err)
	}

	got, ok := decodeConversationIndexFromRaw(blob)
	if !ok {
		t.Fatal("expected decodeConversationIndexFromRaw to decode []byte payload")
	}
	if got.Version != conversationIndexVersion {
		t.Fatalf("index version = %d, want %d", got.Version, conversationIndexVersion)
	}
	if len(got.Conversations) != 1 {
		t.Fatalf("index conversations len = %d, want 1", len(got.Conversations))
	}
}

func TestDecodeConversationIndexFromRawInvalid(t *testing.T) {
	if _, ok := decodeConversationIndexFromRaw([]byte(`["not","an","index"]`)); ok {
		t.Fatal("expected invalid index payload to fail decode")
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

func TestForceDeterministicCompactionIgnoresTrigger(t *testing.T) {
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
		CompactionTriggerTokens: state.Tokens + 10000, // would block normal compaction
		MaxRecentExchanges:      1,
		SummaryBudgetTokens:     200,
	}
	normal := maybeCompactConversationDeterministic(state, cfg)
	if len(normal.Exchanges) != 3 {
		t.Fatalf("normal deterministic compaction should be noop, got exchanges=%d", len(normal.Exchanges))
	}
	result := forceCompactConversationDeterministic(state, cfg)
	forced := result.State
	older := result.Older
	if len(older) != 2 {
		t.Fatalf("forced compaction older size=%d, want 2", len(older))
	}
	if len(forced.Exchanges) != 1 {
		t.Fatalf("forced compaction exchanges kept=%d, want 1", len(forced.Exchanges))
	}
	if forced.Summary == "" {
		t.Fatal("expected forced compaction to produce a summary")
	}
}

func TestForceDeterministicCompactionCompactsWhenBelowRecentWindow(t *testing.T) {
	state := conversationState{
		Profile: defaultProfile,
		Exchanges: []conversationExchange{
			{Human: "alice says: one", AI: "ai: one"},
			{Human: "alice says: two", AI: "ai: two"},
			{Human: "alice says: three", AI: "ai: three"},
			{Human: "alice says: four", AI: "ai: four"},
		},
	}
	state.Tokens = estimateConversationTokens(state.Exchanges)
	cfg := aiConfig{
		CompactionTriggerTokens: state.Tokens + 10000, // would block normal compaction
		MaxRecentExchanges:      12,                   // above current exchange count
		SummaryBudgetTokens:     200,
	}

	normal := maybeCompactConversationDeterministic(state, cfg)
	if len(normal.Exchanges) != 4 {
		t.Fatalf("normal deterministic compaction should be noop, got exchanges=%d", len(normal.Exchanges))
	}

	result := forceCompactConversationDeterministic(state, cfg)
	forced := result.State
	older := result.Older
	if len(older) != 1 {
		t.Fatalf("forced compaction older size=%d, want 1", len(older))
	}
	if len(forced.Exchanges) != 3 {
		t.Fatalf("forced compaction exchanges kept=%d, want 3", len(forced.Exchanges))
	}
	if forced.Summary == "" {
		t.Fatal("expected forced compaction to produce a summary")
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

func TestModelCompactionUsesRefinerWhenEnabled(t *testing.T) {
	state := conversationState{
		Profile: defaultProfile,
		Exchanges: []conversationExchange{
			{Human: "alice says: one", AI: "ai: one"},
			{Human: "alice says: two", AI: "ai: two"},
			{Human: "alice says: three", AI: "ai: three"},
			{Human: "alice says: four", AI: "ai: four"},
		},
	}
	state.Tokens = estimateConversationTokens(state.Exchanges)
	cfg := aiConfig{
		CompactionTriggerTokens: 1,
		MaxRecentExchanges:      2,
		SummaryBudgetTokens:     200,
		EnableModelCompaction:   true,
	}
	got := maybeCompactConversationWithRefiner(state, cfg, func(existing string, older []conversationExchange, cfg aiConfig) (string, error) {
		if len(older) != 2 {
			t.Fatalf("older len = %d, want 2", len(older))
		}
		return "model refined summary", nil
	})
	if got.Summary != "model refined summary" {
		t.Fatalf("summary = %q, want model refined summary", got.Summary)
	}
	if len(got.Exchanges) != 2 {
		t.Fatalf("recent exchanges kept = %d, want 2", len(got.Exchanges))
	}
}

func TestModelCompactionFallsBackOnRefinerError(t *testing.T) {
	state := conversationState{
		Profile: defaultProfile,
		Exchanges: []conversationExchange{
			{Human: "alice says: one", AI: "ai: one"},
			{Human: "alice says: two", AI: "ai: two"},
			{Human: "alice says: three", AI: "ai: three"},
			{Human: "alice says: four", AI: "ai: four"},
		},
	}
	state.Tokens = estimateConversationTokens(state.Exchanges)
	cfg := aiConfig{
		CompactionTriggerTokens: 1,
		MaxRecentExchanges:      2,
		SummaryBudgetTokens:     200,
		EnableModelCompaction:   true,
	}
	deterministic := maybeCompactConversationDeterministic(state, cfg)
	got := maybeCompactConversationWithRefiner(state, cfg, func(existing string, older []conversationExchange, cfg aiConfig) (string, error) {
		return "", errors.New("synthetic failure")
	})
	if got.Summary != deterministic.Summary {
		t.Fatalf("fallback summary mismatch: got %q want %q", got.Summary, deterministic.Summary)
	}
}

func TestModelCompactionDisabledDoesNotCallRefiner(t *testing.T) {
	state := conversationState{
		Profile: defaultProfile,
		Exchanges: []conversationExchange{
			{Human: "alice says: one", AI: "ai: one"},
			{Human: "alice says: two", AI: "ai: two"},
			{Human: "alice says: three", AI: "ai: three"},
			{Human: "alice says: four", AI: "ai: four"},
		},
	}
	state.Tokens = estimateConversationTokens(state.Exchanges)
	cfg := aiConfig{
		CompactionTriggerTokens: 1,
		MaxRecentExchanges:      2,
		SummaryBudgetTokens:     200,
		EnableModelCompaction:   false,
	}
	called := false
	got := maybeCompactConversationWithRefiner(state, cfg, func(existing string, older []conversationExchange, cfg aiConfig) (string, error) {
		called = true
		return "should-not-happen", nil
	})
	if called {
		t.Fatal("refiner should not be called when model compaction is disabled")
	}
	if got.Summary == "" {
		t.Fatal("expected deterministic summary to still be produced")
	}
}

func TestChatCompletionsEndpoint(t *testing.T) {
	tests := []struct {
		name string
		cfg  aiConfig
		want string
	}{
		{
			name: "no default endpoint",
			cfg:  aiConfig{},
			want: "",
		},
		{
			name: "openai compatible base url",
			cfg: aiConfig{
				APIBaseURL: "https://generativelanguage.googleapis.com/v1beta/openai/",
			},
			want: "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		},
		{
			name: "base url already points at chat completions",
			cfg: aiConfig{
				APIBaseURL: "https://example.test/openai/chat/completions",
			},
			want: "https://example.test/openai/chat/completions",
		},
		{
			name: "exact endpoint overrides base url",
			cfg: aiConfig{
				APIBaseURL:         "https://example.test/openai/",
				ChatCompletionsURL: "https://proxy.example.test/v1/chat/completions",
			},
			want: "https://proxy.example.test/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatCompletionsEndpoint(tt.cfg); got != tt.want {
				t.Fatalf("chatCompletionsEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAIProviderName(t *testing.T) {
	if got := aiProviderName(aiConfig{}); got != "AI provider" {
		t.Fatalf("aiProviderName(empty) = %q, want AI provider", got)
	}
	if got := aiProviderName(aiConfig{ProviderName: " Gemini "}); got != "Gemini" {
		t.Fatalf("aiProviderName(Gemini) = %q, want Gemini", got)
	}
}

func TestFriendlyAIErrorUsesProviderName(t *testing.T) {
	got := friendlyAIError("Gemini", http.StatusUnauthorized, "401 Unauthorized", nil)
	if !strings.Contains(got, "Gemini authentication failed") {
		t.Fatalf("friendlyAIError() = %q, want Gemini authentication message", got)
	}
}
