package main

import "testing"

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
