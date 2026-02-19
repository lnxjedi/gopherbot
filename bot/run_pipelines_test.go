package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestInheritChildJobProtocolUsesIncomingProtocol(t *testing.T) {
	parent := &worker{
		Protocol: robot.Slack,
		Incoming: &robot.ConnectorMessage{
			Protocol: "ssh",
		},
	}
	child := &worker{
		Protocol: robot.Test,
		Incoming: &robot.ConnectorMessage{},
	}

	inheritChildJobProtocol(parent, child)

	if child.Protocol != robot.SSH {
		t.Fatalf("child.Protocol = %v, want %v", child.Protocol, robot.SSH)
	}
	if child.Incoming == nil || child.Incoming.Protocol != "ssh" {
		t.Fatalf("child.Incoming.Protocol = %#v, want %q", child.Incoming, "ssh")
	}
}

func TestInheritChildJobProtocolFallsBackToParentEnum(t *testing.T) {
	parent := &worker{
		Protocol: robot.Terminal,
		Incoming: &robot.ConnectorMessage{},
	}
	child := &worker{}

	inheritChildJobProtocol(parent, child)

	if child.Protocol != robot.Terminal {
		t.Fatalf("child.Protocol = %v, want %v", child.Protocol, robot.Terminal)
	}
	if child.Incoming == nil || child.Incoming.Protocol != "terminal" {
		t.Fatalf("child.Incoming.Protocol = %#v, want %q", child.Incoming, "terminal")
	}
}

func TestInheritChildJobProtocolNilSafe(t *testing.T) {
	inheritChildJobProtocol(nil, nil)
	inheritChildJobProtocol(&worker{}, nil)
	inheritChildJobProtocol(nil, &worker{})
}
