package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func resetAIDevCommandsForTest(user string, prefix byte, consume bool) {
	aidevCommands.Lock()
	defer aidevCommands.Unlock()

	aidevCommands.enabled = user != ""
	aidevCommands.user = user
	aidevCommands.prefix = prefix
	aidevCommands.consume = consume
	aidevCommands.buffer = make([]aidevCommandEvent, defaultAIDevCommandBufferSize)
	aidevCommands.bufIdx = 0
	aidevCommands.filled = false
	aidevCommands.nextSeq = 0
	aidevCommands.waiters = map[chan struct{}]struct{}{}
}

func TestAIDevCommandCaptureAndFilter(t *testing.T) {
	resetAIDevCommandsForTest("david", '>', true)

	captured := captureAIDevCommandIfMatched("david", &robot.ConnectorMessage{
		Protocol:    "ssh",
		UserID:      "U001",
		ChannelName: "general",
		ThreadID:    "t-1",
		MessageID:   "m-1",
	}, true, ">what is the status?")
	if !captured {
		t.Fatal("expected command to be captured")
	}

	all := getAIDevCommands(aidevCommandQuery{All: true, Limit: 10, TimeoutMS: 0})
	if len(all.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(all.Commands))
	}
	if all.Commands[0].Command != "what is the status?" {
		t.Fatalf("unexpected command payload: %q", all.Commands[0].Command)
	}
	if all.NextCursor != 1 || all.Latest != 1 {
		t.Fatalf("unexpected cursors: next=%d latest=%d", all.NextCursor, all.Latest)
	}

	filtered := getAIDevCommands(aidevCommandQuery{AfterCursor: 1, Limit: 10, TimeoutMS: 0})
	if len(filtered.Commands) != 0 {
		t.Fatalf("expected no commands after cursor 1, got %d", len(filtered.Commands))
	}
}

func TestAIDevCommandCaptureRejectsNonMatching(t *testing.T) {
	resetAIDevCommandsForTest("david", '>', true)

	tests := []struct {
		name      string
		user      string
		isCommand bool
		msg       string
	}{
		{name: "wrong user", user: "alice", isCommand: true, msg: ">hello"},
		{name: "not command", user: "david", isCommand: false, msg: ">hello"},
		{name: "missing prefix", user: "david", isCommand: true, msg: "hello"},
	}

	for _, tt := range tests {
		if captured := captureAIDevCommandIfMatched(tt.user, &robot.ConnectorMessage{Protocol: "ssh"}, tt.isCommand, tt.msg); captured {
			t.Fatalf("%s: expected no capture", tt.name)
		}
	}

	batch := getAIDevCommands(aidevCommandQuery{All: true, TimeoutMS: 0})
	if len(batch.Commands) != 0 {
		t.Fatalf("expected no commands, got %d", len(batch.Commands))
	}
}

func TestAIDevCommandConsumeFlag(t *testing.T) {
	resetAIDevCommandsForTest("david", '>', true)
	if !aidevCommandConduitConsumes() {
		t.Fatal("expected consume=true")
	}

	resetAIDevCommandsForTest("david", '>', false)
	if aidevCommandConduitConsumes() {
		t.Fatal("expected consume=false")
	}
}
