package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func setRuntimeBotIDTestState(t *testing.T, primary, defaultProtocol string, ids map[string]string) {
	t.Helper()

	runtimeConnectors.Lock()
	oldPrimary := runtimeConnectors.primary
	oldDefault := runtimeConnectors.defaultProtocol
	oldIDs := copyStringMap(runtimeConnectors.botIDs)
	runtimeConnectors.Unlock()

	runtimeConnectors.Lock()
	runtimeConnectors.primary = primary
	runtimeConnectors.defaultProtocol = defaultProtocol
	runtimeConnectors.botIDs = copyStringMap(ids)
	runtimeConnectors.Unlock()

	t.Cleanup(func() {
		runtimeConnectors.Lock()
		runtimeConnectors.primary = oldPrimary
		runtimeConnectors.defaultProtocol = oldDefault
		runtimeConnectors.botIDs = oldIDs
		runtimeConnectors.Unlock()
	})
}

func TestConnectorHandlerSetBotIDStoresPerProtocol(t *testing.T) {
	setRuntimeBotIDTestState(t, "slack", "slack", map[string]string{})

	secondary := connectorHandler{handler: handle, protocol: "ssh", allowBotIdentity: false}
	secondary.SetBotID("ssh-ed25519 AAAABBBB")

	sshID, ok := getRuntimeBotID("ssh")
	if !ok || sshID != "ssh-ed25519 AAAABBBB" {
		t.Fatalf("getRuntimeBotID(\"ssh\") = %q, %t; want %q, true", sshID, ok, "ssh-ed25519 AAAABBBB")
	}

	primary := connectorHandler{handler: handle, protocol: "slack", allowBotIdentity: false}
	primary.SetBotID("U12345")

	slackID, ok := getRuntimeBotID("slack")
	if !ok || slackID != "U12345" {
		t.Fatalf("getRuntimeBotID(\"slack\") = %q, %t; want %q, true", slackID, ok, "U12345")
	}
}

func TestGetBotAttributeIDUsesIncomingProtocol(t *testing.T) {
	setRuntimeBotIDTestState(t, "slack", "slack", map[string]string{
		"slack": "U12345",
		"ssh":   "ssh-ed25519 AAAABBBB",
	})

	r := Robot{
		Message: &robot.Message{
			Protocol: robot.SSH,
			Incoming: &robot.ConnectorMessage{Protocol: "ssh"},
		},
	}

	got := r.GetBotAttribute("id").String()
	if got != "<ssh-ed25519 AAAABBBB>" {
		t.Fatalf("GetBotAttribute(\"id\") = %q, want %q", got, "<ssh-ed25519 AAAABBBB>")
	}
}

func TestGetBotAttributeIDFallsBackToDefaultProtocol(t *testing.T) {
	setRuntimeBotIDTestState(t, "slack", "slack", map[string]string{
		"slack": "U12345",
	})

	r := Robot{
		Message: &robot.Message{
			Protocol: robot.Test,
			Incoming: &robot.ConnectorMessage{},
		},
	}

	got := r.GetBotAttribute("id").String()
	if got != "<U12345>" {
		t.Fatalf("GetBotAttribute(\"id\") = %q, want %q", got, "<U12345>")
	}
}

func TestGetBotAttributeIDReturnsEmptyWithoutRuntimeBotID(t *testing.T) {
	setRuntimeBotIDTestState(t, "slack", "slack", map[string]string{})

	r := Robot{
		Message: &robot.Message{
			Protocol: robot.Slack,
			Incoming: &robot.ConnectorMessage{Protocol: "slack"},
		},
	}

	got := r.GetBotAttribute("id").String()
	if got != "<>" {
		t.Fatalf("GetBotAttribute(\"id\") = %q, want %q", got, "<>")
	}
}
