package slack

import (
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
)

func TestSlackifyMessageVariableUsesRichTextBlocks(t *testing.T) {
	s := &slackConnector{maxMessageSplit: 1}

	msgs := s.slackifyMessage("", "", "", "literal @here and https://example.com", robot.Variable, &robot.ConnectorMessage{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if got := msgs[0].text; got != "literal @here and https://example.com" {
		t.Fatalf("payload text = %q", got)
	}

	block, ok := msgs[0].blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", msgs[0].blocks[0])
	}
	if len(block.Elements) != 1 {
		t.Fatalf("expected 1 rich text element, got %d", len(block.Elements))
	}

	section, ok := block.Elements[0].(*slack.RichTextSection)
	if !ok {
		t.Fatalf("expected RichTextSection, got %T", block.Elements[0])
	}
	if section.Type != slack.RTESection {
		t.Fatalf("expected rich text section type, got %q", section.Type)
	}
	if len(section.Elements) != 1 {
		t.Fatalf("expected 1 section element, got %d", len(section.Elements))
	}

	textElem, ok := section.Elements[0].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected text element, got %T", section.Elements[0])
	}
	if textElem.Text != "literal @here and https://example.com" {
		t.Fatalf("rich text section text = %q", textElem.Text)
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

	block, ok := msgs[0].blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", msgs[0].blocks[0])
	}
	section, ok := block.Elements[0].(*slack.RichTextSection)
	if !ok {
		t.Fatalf("expected RichTextSection, got %T", block.Elements[0])
	}
	textElem, ok := section.Elements[0].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected text element, got %T", section.Elements[0])
	}
	if got := textElem.Text; got != "@alice: hello" {
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
	first, ok := msgs[0].blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", msgs[0].blocks[0])
	}
	firstSection, ok := first.Elements[0].(*slack.RichTextSection)
	if !ok {
		t.Fatalf("expected RichTextSection, got %T", first.Elements[0])
	}
	firstText, ok := firstSection.Elements[0].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected text element, got %T", firstSection.Elements[0])
	}
	if len(firstText.Text) != slackBlockTextLimit {
		t.Fatalf("expected first block chunk length %d, got %d", slackBlockTextLimit, len(firstText.Text))
	}
	second, ok := msgs[1].blocks[0].(*slack.RichTextBlock)
	if !ok {
		t.Fatalf("expected RichTextBlock, got %T", msgs[1].blocks[0])
	}
	secondSection, ok := second.Elements[0].(*slack.RichTextSection)
	if !ok {
		t.Fatalf("expected RichTextSection, got %T", second.Elements[0])
	}
	secondText, ok := secondSection.Elements[0].(*slack.RichTextSectionTextElement)
	if !ok {
		t.Fatalf("expected text element, got %T", secondSection.Elements[0])
	}
	if len(secondText.Text) != 10 {
		t.Fatalf("expected second block chunk length 10, got %d", len(secondText.Text))
	}
}
