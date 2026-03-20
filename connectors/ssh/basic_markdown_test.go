package ssh

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestRenderBasicMarkdownPlain(t *testing.T) {
	in := "**Deploy status:** *rollback in progress*\nSee [runbook](https://example.com/runbook)\nEscaped: \\*not bold\\* and \\`not code\\` and \\@alice\nInline: `kubectl get pods`"
	got := renderBasicMarkdownPlain(in)
	want := "Deploy status: rollback in progress\nSee runbook (https://example.com/runbook)\nEscaped: *not bold* and `not code` and @alice\nInline: kubectl get pods"
	if got != want {
		t.Fatalf("renderBasicMarkdownPlain() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownPlainFencedCode(t *testing.T) {
	in := "Before\n```yaml\napiVersion: v1\nkind: Pod\n```\nAfter"
	got := renderBasicMarkdownPlain(in)
	want := "Before\n\napiVersion: v1\nkind: Pod\n\nAfter"
	if got != want {
		t.Fatalf("renderBasicMarkdownPlain() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownPlainEmoji(t *testing.T) {
	in := "Build passed :white_check_mark: but :custom_shipit: stays literal"
	got := renderBasicMarkdownPlain(in)
	want := "Build passed \u2705 but :custom_shipit: stays literal"
	if got != want {
		t.Fatalf("renderBasicMarkdownPlain() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownPlainEmojiNotParsedInCode(t *testing.T) {
	in := "Inline `:joy:`\n```txt\n:rocket:\n```\nDone :rocket:"
	got := renderBasicMarkdownPlain(in)
	want := "Inline :joy:\n\n:rocket:\n\nDone \U0001f680"
	if got != want {
		t.Fatalf("renderBasicMarkdownPlain() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownPlainEmojiLinkLabel(t *testing.T) {
	in := "See [:eyes: runbook](https://example.com/runbook)"
	got := renderBasicMarkdownPlain(in)
	want := "See \U0001f440 runbook (https://example.com/runbook)"
	if got != want {
		t.Fatalf("renderBasicMarkdownPlain() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownPlainEmojiMalformedToken(t *testing.T) {
	in := "Keep abc:joy: and http://example.com and :rocket:"
	got := renderBasicMarkdownPlain(in)
	want := "Keep abc:joy: and http://example.com and \U0001f680"
	if got != want {
		t.Fatalf("renderBasicMarkdownPlain() = %q, want %q", got, want)
	}
}

func TestSendProtocolChannelThreadMessageBasicMarkdown(t *testing.T) {
	sc := &sshConnector{
		cfg:     sshConfig{DefaultChannel: "general"},
		botName: "floyd",
		botID:   "botid",
		buffer:  make([]bufferMsg, 8),
		clients: make(map[*sshClient]struct{}),
		threads: make(map[string]int),
		waiters: make(map[chan struct{}]struct{}),
	}

	msg := "**Deploy status:** *rollback in progress* :white_check_mark:"
	if ret := sc.SendProtocolChannelThreadMessage("general", "", msg, robot.BasicMarkdown, nil); ret != robot.Ok {
		t.Fatalf("SendProtocolChannelThreadMessage() ret = %v, want %v", ret, robot.Ok)
	}

	snap := sc.snapshotBuffer()
	if len(snap) != 1 {
		t.Fatalf("expected 1 buffered message, got %d", len(snap))
	}
	if snap[0].text != "Deploy status: rollback in progress \u2705" {
		t.Fatalf("buffered text = %q", snap[0].text)
	}
}

func TestSendProtocolChannelThreadMessageRawUnchanged(t *testing.T) {
	sc := &sshConnector{
		cfg:     sshConfig{DefaultChannel: "general"},
		botName: "floyd",
		botID:   "botid",
		buffer:  make([]bufferMsg, 8),
		clients: make(map[*sshClient]struct{}),
		threads: make(map[string]int),
		waiters: make(map[chan struct{}]struct{}),
	}

	msg := "**Deploy status:** *rollback in progress*"
	if ret := sc.SendProtocolChannelThreadMessage("general", "", msg, robot.Raw, nil); ret != robot.Ok {
		t.Fatalf("SendProtocolChannelThreadMessage() ret = %v, want %v", ret, robot.Ok)
	}

	snap := sc.snapshotBuffer()
	if len(snap) != 1 {
		t.Fatalf("expected 1 buffered message, got %d", len(snap))
	}
	if snap[0].text != msg {
		t.Fatalf("buffered text = %q, want %q", snap[0].text, msg)
	}
}
