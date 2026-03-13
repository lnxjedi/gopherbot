package slack

import (
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
)

func TestSlackifyMessageVariableUsesPlainTextBlocks(t *testing.T) {
	s := &slackConnector{maxMessageSplit: 1}

	msgs := s.slackifyMessage("", "", "", "literal @here and https://example.com", robot.Variable, &robot.ConnectorMessage{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if got := msgs[0].text; got != "literal @here and https://example.com" {
		t.Fatalf("payload text = %q", got)
	}

	block, ok := msgs[0].blocks[0].(*slack.SectionBlock)
	if !ok {
		t.Fatalf("expected SectionBlock, got %T", msgs[0].blocks[0])
	}
	if block.Text == nil {
		t.Fatalf("expected text object on section block")
	}
	if block.Text.Type != slack.PlainTextType {
		t.Fatalf("expected plain_text block, got %q", block.Text.Type)
	}
	if block.Text.Text != "literal @here and https://example.com" {
		t.Fatalf("section text = %q", block.Text.Text)
	}
	if block.Text.Verbatim {
		t.Fatalf("plain_text block should not set verbatim")
	}
}

func TestSlackifyMessageFixedUsesPreformattedRichTextBlock(t *testing.T) {
	s := &slackConnector{maxMessageSplit: 1}

	msgs := s.slackifyMessage("", "", "", "line 1\nline 2", robot.Fixed, &robot.ConnectorMessage{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if got := msgs[0].text; got != "line 1\nline 2" {
		t.Fatalf("payload text = %q", got)
	}

	block, ok := msgs[0].blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", msgs[0].blocks[0])
	}
	if len(block.Elements) != 1 {
		t.Fatalf("expected 1 rich text element, got %d", len(block.Elements))
	}

	pre, ok := block.Elements[0].(*slack.RichTextPreformatted)
	if !ok {
		t.Fatalf("expected RichTextPreformatted, got %T", block.Elements[0])
	}
	if pre.Type != slack.RTEPreformatted {
		t.Fatalf("expected preformatted type, got %q", pre.Type)
	}
	if len(pre.Elements) != 1 {
		t.Fatalf("expected 1 preformatted element, got %d", len(pre.Elements))
	}

	textElem, ok := pre.Elements[0].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected text element, got %T", pre.Elements[0])
	}
	if textElem.Text != "line 1\nline 2" {
		t.Fatalf("preformatted text = %q", textElem.Text)
	}
}

func TestSlackifyMessageVariableUsesReadableBlockPrefix(t *testing.T) {
	s := &slackConnector{maxMessageSplit: 1}

	msgs := s.slackifyMessage("U123", "<@U123>: ", "@alice: ", "hello", robot.Variable, &robot.ConnectorMessage{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	block, ok := msgs[0].blocks[0].(*slack.SectionBlock)
	if !ok {
		t.Fatalf("expected SectionBlock, got %T", msgs[0].blocks[0])
	}
	if got := block.Text.Text; got != "@alice: hello" {
		t.Fatalf("section text = %q", got)
	}
	if !strings.Contains(msgs[0].legacyText, "@alice") {
		t.Fatalf("legacy text should keep readable user prefix, got %q", msgs[0].legacyText)
	}
}

func TestSlackifyMessageVariableSplitsToBlockLimit(t *testing.T) {
	s := &slackConnector{maxMessageSplit: 2}

	msg := strings.Repeat("a", slackBlockTextLimit+10)
	msgs := s.slackifyMessage("", "", "", msg, robot.Variable, &robot.ConnectorMessage{})
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	first, ok := msgs[0].blocks[0].(*slack.SectionBlock)
	if !ok {
		t.Fatalf("expected SectionBlock, got %T", msgs[0].blocks[0])
	}
	if len(first.Text.Text) != slackBlockTextLimit {
		t.Fatalf("expected first block chunk length %d, got %d", slackBlockTextLimit, len(first.Text.Text))
	}
	second, ok := msgs[1].blocks[0].(*slack.SectionBlock)
	if !ok {
		t.Fatalf("expected SectionBlock, got %T", msgs[1].blocks[0])
	}
	if len(second.Text.Text) != 10 {
		t.Fatalf("expected second block chunk length 10, got %d", len(second.Text.Text))
	}
}
