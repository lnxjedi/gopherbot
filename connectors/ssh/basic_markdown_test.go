package ssh

import (
	"strings"
	"testing"
	"time"

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

func TestRenderBasicMarkdownStyled(t *testing.T) {
	in := "**Deploy status:** *rollback in progress* `kubectl get pods` :rocket:\nSee [runbook](https://example.com/runbook)"
	got := renderBasicMarkdownStyled(in, basicMarkdownStyle{})
	want := "\x1b[1mDeploy status:\x1b[22m \x1b[3mrollback in progress\x1b[23m kubectl get pods \U0001f680\nSee runbook (https://example.com/runbook)"
	if got != want {
		t.Fatalf("renderBasicMarkdownStyled() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownStyledDoesNotParseCode(t *testing.T) {
	in := "Inline `**stay** :joy:`\n```txt\n*still literal* :rocket:\n```\nDone *now*"
	got := renderBasicMarkdownStyled(in, basicMarkdownStyle{})
	want := "Inline **stay** :joy:\n\n*still literal* :rocket:\n\nDone \x1b[3mnow\x1b[23m"
	if got != want {
		t.Fatalf("renderBasicMarkdownStyled() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownStyledCodeColors(t *testing.T) {
	in := "Run `kubectl get pods`\n```txt\nstatus=ok\n```\nDone"
	got := renderBasicMarkdownStyled(in, basicMarkdownStyle{
		inlineCodeANSI: [2]string{"\x1b[38;5;210m", "\x1b[38;5;81m"},
		codeBlockANSI:  [2]string{"\x1b[38;5;48m", "\x1b[38;5;81m"},
	})
	want := "Run \x1b[38;5;210mkubectl get pods\x1b[38;5;81m\n\x1b[38;5;48m\nstatus=ok\n\x1b[38;5;81m\nDone"
	if got != want {
		t.Fatalf("renderBasicMarkdownStyled() = %q, want %q", got, want)
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

func TestFormatMessageBasicMarkdownUsesStyledDisplaySource(t *testing.T) {
	client := &sshClient{
		userName: "alice",
		channel:  "general",
		width:    200,
		color:    true,
		colorScheme: map[string]int{
			"prompt":     39,
			"timestamp":  244,
			"bot":        81,
			"user":       114,
			"inlinecode": 210,
			"codeblock":  48,
		},
	}
	evt := bufferMsg{
		timestamp:           time.Date(2026, time.March, 20, 9, 15, 45, 0, time.UTC),
		userName:            "floyd",
		userID:              "botid",
		isBot:               true,
		channel:             "general",
		text:                "Deploy status: rollback in progress\nstatus=ok",
		basicMarkdownSource: "**Deploy status:** *rollback in progress* `kubectl get pods`\n```txt\nstatus=ok\n```",
	}

	got := client.formatMessage(evt, false, false)
	if !strings.Contains(got, "\x1b[1mDeploy status:\x1b[22m") {
		t.Fatalf("formatted message missing bold ANSI: %q", got)
	}
	if !strings.Contains(got, "\x1b[3mrollback in progress\x1b[23m") {
		t.Fatalf("formatted message missing italic ANSI: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;5;210mkubectl get pods\x1b[38;5;81m") {
		t.Fatalf("formatted message missing inline code ANSI: %q", got)
	}
	if !strings.Contains(got, "\x1b[38;5;48m\nstatus=ok\n\x1b[38;5;81m") {
		t.Fatalf("formatted message missing code block ANSI: %q", got)
	}
	if !strings.Contains(got, "09:15:45") {
		t.Fatalf("formatted message missing timestamp: %q", got)
	}
}

func TestFormatMessageFixedMultilineStartsBodyAtColumnZero(t *testing.T) {
	client := &sshClient{
		userName: "alice",
		channel:  "general",
		width:    20,
	}
	evt := bufferMsg{
		timestamp: time.Date(2026, time.March, 20, 9, 15, 45, 0, time.UTC),
		userName:  "floyd",
		userID:    "botid",
		isBot:     true,
		channel:   "general",
		text:      "PID COMMAND\n1   init",
		fixed:     true,
	}

	got := client.formatMessage(evt, false, false)
	want := "(09:15:45)=@floyd/#general:\nPID COMMAND\n1   init"
	if got != want {
		t.Fatalf("formatMessage() = %q, want %q", got, want)
	}
}

func TestPrepareSSHDisplayMessageDoesNotInjectFixedNewline(t *testing.T) {
	got, markdownSource := prepareSSHDisplayMessage("PID COMMAND\n1   init", robot.Fixed)
	if got != "PID COMMAND\n1   init" {
		t.Fatalf("prepareSSHDisplayMessage() plain = %q", got)
	}
	if markdownSource != "" {
		t.Fatalf("prepareSSHDisplayMessage() markdownSource = %q, want empty", markdownSource)
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
