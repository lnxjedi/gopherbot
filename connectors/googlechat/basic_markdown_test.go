package googlechat

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestRenderBasicMarkdownConvertsChatTextSyntax(t *testing.T) {
	gc := &googleChatConnector{
		botUserMap: map[string]string{"alice": "users/123"},
	}

	in := "**bold** *italic* [Example](https://example.com) @alice"
	got := gc.renderMessageText(in, robot.BasicMarkdown)
	want := "*bold* _italic_ <https://example.com|Example> <users/123>"
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
