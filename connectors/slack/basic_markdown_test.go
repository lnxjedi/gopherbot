package slack

import "testing"

func TestRenderBasicMarkdownMentions(t *testing.T) {
	s := &slackConnector{
		userMap: map[string]string{
			"alice": "U111",
		},
	}

	in := "Paging @alice and @unknown. Email foo@example.com."
	got := s.renderBasicMarkdown(in)
	want := "Paging <@U111> and @unknown. Email foo@example.com."
	if got != want {
		t.Fatalf("renderBasicMarkdown() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownCodeBoundaries(t *testing.T) {
	s := &slackConnector{
		userMap: map[string]string{
			"alice": "U111",
		},
	}

	in := "inline `@alice` and block:\n```text\n@alice :white_check_mark:\n```\noutside @alice"
	got := s.renderBasicMarkdown(in)
	want := "inline `@alice` and block:\n```\n@alice :white_check_mark:\n```\noutside <@U111>"
	if got != want {
		t.Fatalf("renderBasicMarkdown() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownLinksAndEscapes(t *testing.T) {
	s := &slackConnector{
		userMap: map[string]string{
			"alice": "U111",
		},
	}

	in := "See [runbook](https://example.com/runbook) and \\[literal\\](https://example.com) and \\@alice"
	got := s.renderBasicMarkdown(in)
	want := "See <https://example.com/runbook|runbook> and [literal](https://example.com) and @alice"
	if got != want {
		t.Fatalf("renderBasicMarkdown() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownEmojiPassThrough(t *testing.T) {
	s := &slackConnector{}
	in := "Build passed :white_check_mark: 😂"
	got := s.renderBasicMarkdown(in)
	if got != in {
		t.Fatalf("renderBasicMarkdown() = %q, want %q", got, in)
	}
}

func TestRenderBasicMarkdownCaseInsensitiveMention(t *testing.T) {
	s := &slackConnector{
		userMap: map[string]string{
			"alice": "U111",
		},
	}
	in := "Please review @ALICE"
	got := s.renderBasicMarkdown(in)
	want := "Please review <@U111>"
	if got != want {
		t.Fatalf("renderBasicMarkdown() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownAmbiguousCaseMentionStaysLiteral(t *testing.T) {
	s := &slackConnector{
		userMap: map[string]string{
			"alice": "U111",
			"ALICE": "U222",
		},
	}
	in := "Please review @AlIcE"
	got := s.renderBasicMarkdown(in)
	want := "Please review @AlIcE"
	if got != want {
		t.Fatalf("renderBasicMarkdown() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownEmphasisToSlack(t *testing.T) {
	s := &slackConnector{}
	in := "**Deploy status:** *rollback in progress*"
	got := s.renderBasicMarkdown(in)
	want := "*Deploy status:* _rollback in progress_"
	if got != want {
		t.Fatalf("renderBasicMarkdown() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownEscapedFormattingStaysLiteral(t *testing.T) {
	s := &slackConnector{
		userMap: map[string]string{
			"alice": "U111",
		},
	}

	in := "Escaping: \\*not bold\\* and \\`not code\\` and \\@alice and [label](https://example.com)"
	got := s.renderBasicMarkdown(in)
	want := "Escaping: " + escapePad + "*not bold" + escapePad + "* and " + escapePad + "`not code" + escapePad + "` and @alice and <https://example.com|label>"
	if got != want {
		t.Fatalf("renderBasicMarkdown() = %q, want %q", got, want)
	}
}
