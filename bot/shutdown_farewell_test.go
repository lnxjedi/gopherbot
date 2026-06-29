package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func withShutdownFarewellCapture(t *testing.T) *startupGateCaptureConnector {
	t.Helper()
	conn := &startupGateCaptureConnector{}
	oldConnector := interfaces.Connector
	interfaces.Connector = conn
	state.Lock()
	oldFarewell := state.pendingShutdownFarewell
	state.pendingShutdownFarewell = nil
	state.Unlock()
	t.Cleanup(func() {
		interfaces.Connector = oldConnector
		state.Lock()
		state.pendingShutdownFarewell = oldFarewell
		state.Unlock()
	})
	return conn
}

func TestPendingShutdownFarewellUsesReplyContext(t *testing.T) {
	conn := withShutdownFarewellCapture(t)
	msgObject := &robot.ConnectorMessage{
		Protocol:        "slack",
		ThreadID:        "T123",
		ThreadedMessage: true,
	}
	r := Robot{Message: &robot.Message{
		User:            "alice",
		ProtocolUser:    "U123",
		Channel:         "general",
		ProtocolChannel: "C123",
		Incoming:        msgObject,
		Format:          robot.BasicMarkdown,
	}}

	queueShutdownFarewell(r, "Sayonara!")
	sendPendingShutdownFarewell()

	if conn.sends != 1 {
		t.Fatalf("sends = %d, want 1", conn.sends)
	}
	if conn.user != "U123" || conn.username != "alice" {
		t.Fatalf("reply user = %q/%q, want U123/alice", conn.user, conn.username)
	}
	if conn.channel != "C123" || conn.thread != "T123" {
		t.Fatalf("reply destination = %q/%q, want C123/T123", conn.channel, conn.thread)
	}
	if conn.msg != "Sayonara!" {
		t.Fatalf("reply message = %q, want Sayonara!", conn.msg)
	}
	if conn.format != robot.BasicMarkdown {
		t.Fatalf("reply format = %v, want BasicMarkdown", conn.format)
	}
	if conn.protocol != "slack" {
		t.Fatalf("reply protocol = %q, want slack", conn.protocol)
	}

	sendPendingShutdownFarewell()
	if conn.sends != 1 {
		t.Fatalf("second send count = %d, want one-shot send", conn.sends)
	}
}

func TestPendingShutdownFarewellUsesDirectMessageContext(t *testing.T) {
	conn := withShutdownFarewellCapture(t)
	r := Robot{Message: &robot.Message{
		User:         "alice",
		ProtocolUser: "U123",
		Incoming:     &robot.ConnectorMessage{Protocol: "ssh"},
		Format:       robot.Raw,
	}}

	queueShutdownFarewell(r, "Later")
	sendPendingShutdownFarewell()

	if conn.sends != 1 {
		t.Fatalf("sends = %d, want 1", conn.sends)
	}
	if conn.user != "U123" {
		t.Fatalf("direct user = %q, want U123", conn.user)
	}
	if conn.channel != "" || conn.thread != "" {
		t.Fatalf("direct destination = %q/%q, want empty channel/thread", conn.channel, conn.thread)
	}
	if conn.msg != "Later" {
		t.Fatalf("direct message = %q, want Later", conn.msg)
	}
	if conn.format != robot.Raw {
		t.Fatalf("direct format = %v, want Raw", conn.format)
	}
	if conn.protocol != "ssh" {
		t.Fatalf("direct protocol = %q, want ssh", conn.protocol)
	}
}
