package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestWorkerCloneCopiesIncomingByValue(t *testing.T) {
	orig := &worker{
		Incoming: &robot.ConnectorMessage{
			Protocol:        "ssh",
			ThreadID:        "thread-a",
			ThreadedMessage: true,
		},
	}

	clone := orig.clone()
	if clone.Incoming == nil {
		t.Fatalf("clone.Incoming is nil")
	}
	if clone.Incoming == orig.Incoming {
		t.Fatalf("clone.Incoming points to original Incoming")
	}

	clone.Incoming.ThreadID = "thread-b"
	clone.Incoming.Protocol = "slack"
	clone.Incoming.ThreadedMessage = false

	if orig.Incoming.ThreadID != "thread-a" {
		t.Fatalf("orig.Incoming.ThreadID = %q, want %q", orig.Incoming.ThreadID, "thread-a")
	}
	if orig.Incoming.Protocol != "ssh" {
		t.Fatalf("orig.Incoming.Protocol = %q, want %q", orig.Incoming.Protocol, "ssh")
	}
	if !orig.Incoming.ThreadedMessage {
		t.Fatalf("orig.Incoming.ThreadedMessage = false, want true")
	}
}
