package googlechat

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestRenderBasicMarkdownConvertsChatTextSyntax(t *testing.T) {
	gc := &googleChatConnector{
		botUserMap: map[string]string{"alice": "users/123"},
	}

	in := "**bold** *italic* [Example](https://example.com) @alice :rocket:"
	got := gc.renderMessageText(in, robot.BasicMarkdown)
	want := "*bold* _italic_ <https://example.com|Example> <users/123> \U0001f680"
	if got != want {
		t.Fatalf("renderMessageText() = %q, want %q", got, want)
	}
}

func TestRenderVariableLiteralizesChatFormatting(t *testing.T) {
	gc := &googleChatConnector{}
	in := "*italic* _also_ ~gone~ `code` <https://cnn.com|*CNN*> https://cnn.com\n- list\n> quote"
	got := gc.renderMessageText(in, robot.Variable)
	want := "\u2217italic\u2217 \uff3falso\uff3f \uff5egone\uff5e \uff40code\uff40 \uff1chttps://cnn.com|∗CNN∗\uff1e https://cnn.com\n\u2010 list\n\uff1e quote"
	if got != want {
		t.Fatalf("renderMessageText() = %q, want %q", got, want)
	}
}

func TestRenderFixedLiteralizesInnerFormatting(t *testing.T) {
	gc := &googleChatConnector{}
	in := "<https://cnn.com|*CNN*>"
	got := gc.renderMessageText(in, robot.Fixed)
	want := "```\n" + googleChatZWSP + "<" + googleChatZWSP + "https" + googleChatZWSP + ":" + googleChatZWSP + "/" + googleChatZWSP + "/" + googleChatZWSP + "cnn" + googleChatZWSP + "." + googleChatZWSP + "com" + googleChatZWSP + "|" + googleChatZWSP + "*CNN*" + googleChatZWSP + ">" + googleChatZWSP + "\n```"
	if got != want {
		t.Fatalf("renderMessageText() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownEmojiNotParsedInCode(t *testing.T) {
	gc := &googleChatConnector{}
	in := "Inline `:joy:`\n```txt\n:rocket:\n```\nDone :rocket:"
	got := gc.renderMessageText(in, robot.BasicMarkdown)
	want := "Inline `:joy:`\n```\n:rocket:\n```\nDone \U0001f680"
	if got != want {
		t.Fatalf("renderMessageText() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownCodeLiteralizesFormattingInsideCode(t *testing.T) {
	gc := &googleChatConnector{}
	in := "Inline `*bold* <https://cnn.com|CNN>`\n```txt\n*bold* <https://cnn.com|CNN>\n```"
	got := gc.renderMessageText(in, robot.BasicMarkdown)
	want := "Inline `*bold* " + googleChatZWSP + "<" + googleChatZWSP + "https" + googleChatZWSP + ":" + googleChatZWSP + "/" + googleChatZWSP + "/" + googleChatZWSP + "cnn" + googleChatZWSP + "." + googleChatZWSP + "com" + googleChatZWSP + "|" + googleChatZWSP + "CNN" + googleChatZWSP + ">" + googleChatZWSP + "`\n```\n*bold* " + googleChatZWSP + "<" + googleChatZWSP + "https" + googleChatZWSP + ":" + googleChatZWSP + "/" + googleChatZWSP + "/" + googleChatZWSP + "cnn" + googleChatZWSP + "." + googleChatZWSP + "com" + googleChatZWSP + "|" + googleChatZWSP + "CNN" + googleChatZWSP + ">" + googleChatZWSP + "\n```"
	if got != want {
		t.Fatalf("renderMessageText() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownEmojiLinkLabel(t *testing.T) {
	gc := &googleChatConnector{}
	in := "See [:eyes: runbook](https://example.com/runbook)"
	got := gc.renderMessageText(in, robot.BasicMarkdown)
	want := "See <https://example.com/runbook|\U0001f440 runbook>"
	if got != want {
		t.Fatalf("renderMessageText() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownEscapedFormattingStaysLiteral(t *testing.T) {
	gc := &googleChatConnector{}
	in := "Escaped \\*not bold\\* and \\`not code\\` and [\\*CNN\\*](https://cnn.com)"
	got := gc.renderMessageText(in, robot.BasicMarkdown)
	want := "Escaped " + googleChatZWSP + "*" + googleChatZWSP + "not bold" + googleChatZWSP + "*" + googleChatZWSP + " and " + googleChatZWSP + "`" + googleChatZWSP + "not code" + googleChatZWSP + "`" + googleChatZWSP + " and <https://cnn.com|" + googleChatZWSP + "*" + googleChatZWSP + "CNN" + googleChatZWSP + "*" + googleChatZWSP + ">"
	if got != want {
		t.Fatalf("renderMessageText() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownEmojiMalformedToken(t *testing.T) {
	gc := &googleChatConnector{}
	in := "Keep abc:joy: and http://example.com and :rocket:"
	got := gc.renderMessageText(in, robot.BasicMarkdown)
	want := "Keep abc:joy: and http://example.com and \U0001f680"
	if got != want {
		t.Fatalf("renderMessageText() = %q, want %q", got, want)
	}
}

func TestBuildOutgoingMessageUsesPrivateMessageViewerForHiddenReply(t *testing.T) {
	gc := &googleChatConnector{}
	incoming := &robot.ConnectorMessage{
		HiddenMessage: true,
		ChannelID:     "spaces/AAA",
		UserID:        "users/123",
	}

	message, _ := gc.buildOutgoingMessage("spaces/AAA", "users/123", "", "hello", robot.Variable, incoming)
	if message == nil {
		t.Fatal("buildOutgoingMessage() returned nil")
	}
	if message.PrivateMessageViewer == nil || message.PrivateMessageViewer.Name != "users/123" {
		t.Fatalf("PrivateMessageViewer = %#v", message.PrivateMessageViewer)
	}
	if message.Text != "hello" {
		t.Fatalf("Text = %q, want %q", message.Text, "hello")
	}
}

func TestBuildOutgoingMessageUsesPrivateMessageViewerForHiddenChannelReplyWithoutExplicitUser(t *testing.T) {
	gc := &googleChatConnector{}
	incoming := &robot.ConnectorMessage{
		HiddenMessage: true,
		ChannelID:     "spaces/AAA",
		UserID:        "users/123",
	}

	message, _ := gc.buildOutgoingMessage("spaces/AAA", "", "spaces/AAA/threads/T1", "hello", robot.Variable, incoming)
	if message == nil {
		t.Fatal("buildOutgoingMessage() returned nil")
	}
	if message.PrivateMessageViewer == nil || message.PrivateMessageViewer.Name != "users/123" {
		t.Fatalf("PrivateMessageViewer = %#v", message.PrivateMessageViewer)
	}
	if message.Text != "hello" {
		t.Fatalf("Text = %q, want %q", message.Text, "hello")
	}
}

func TestBuildOutgoingMessageDoesNotUsePrivateMessageViewerWhenHiddenContextUserChanges(t *testing.T) {
	gc := &googleChatConnector{}
	incoming := &robot.ConnectorMessage{
		HiddenMessage: true,
		ChannelID:     "spaces/AAA",
		UserID:        "users/123",
	}

	message, _ := gc.buildOutgoingMessage("spaces/AAA", "users/999", "", "hello", robot.Variable, incoming)
	if message == nil {
		t.Fatal("buildOutgoingMessage() returned nil")
	}
	if message.PrivateMessageViewer != nil {
		t.Fatalf("PrivateMessageViewer = %#v, want nil", message.PrivateMessageViewer)
	}
	if message.Text != "<users/999>: hello" {
		t.Fatalf("Text = %q, want %q", message.Text, "<users/999>: hello")
	}
}

func TestBuildOutgoingMessagePrefixesVisibleDirectedReply(t *testing.T) {
	gc := &googleChatConnector{}
	message, _ := gc.buildOutgoingMessage("spaces/AAA", "users/123", "", "hello", robot.Variable, nil)
	if message == nil {
		t.Fatal("buildOutgoingMessage() returned nil")
	}
	if message.PrivateMessageViewer != nil {
		t.Fatalf("PrivateMessageViewer = %#v, want nil", message.PrivateMessageViewer)
	}
	if message.Text != "<users/123>: hello" {
		t.Fatalf("Text = %q, want %q", message.Text, "<users/123>: hello")
	}
}
