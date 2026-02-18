package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestWorkerMakeMemoryContextUsesUsername(t *testing.T) {
	w := &worker{
		User:     "Parsley",
		Channel:  "general",
		Protocol: robot.SSH,
		Incoming: &robot.ConnectorMessage{
			Protocol:        "ssh",
			UserID:          "ssh-ed25519 AAAA",
			ThreadedMessage: false,
		},
	}

	ctx := w.makeMemoryContext("context:thing")
	if ctx.user != "parsley" {
		t.Fatalf("memory context user = %q, want %q", ctx.user, "parsley")
	}
	if ctx.protocol != "" {
		t.Fatalf("non-thread memory context protocol = %q, want empty", ctx.protocol)
	}
}

func TestWorkerMakeMemoryContextScopesThreadByProtocol(t *testing.T) {
	sshWorker := &worker{
		User:     "parsley",
		Channel:  "general",
		Protocol: robot.SSH,
		Incoming: &robot.ConnectorMessage{
			Protocol:        "ssh",
			ThreadedMessage: true,
			ThreadID:        "0001",
		},
	}
	slackWorker := &worker{
		User:     "parsley",
		Channel:  "general",
		Protocol: robot.Slack,
		Incoming: &robot.ConnectorMessage{
			Protocol:        "slack",
			ThreadedMessage: true,
			ThreadID:        "0001",
		},
	}

	sshCtx := sshWorker.makeMemoryContext("context:thing")
	slackCtx := slackWorker.makeMemoryContext("context:thing")
	if sshCtx == slackCtx {
		t.Fatalf("thread memory contexts should differ across protocols: %+v", sshCtx)
	}
	if sshCtx.protocol != "ssh" {
		t.Fatalf("ssh thread context protocol = %q, want %q", sshCtx.protocol, "ssh")
	}
	if slackCtx.protocol != "slack" {
		t.Fatalf("slack thread context protocol = %q, want %q", slackCtx.protocol, "slack")
	}
}

func TestRobotMakeMemoryContextSharedChannelDropsUser(t *testing.T) {
	r := Robot{
		Message: &robot.Message{
			User:     "parsley",
			Channel:  "general",
			Protocol: robot.SSH,
			Incoming: &robot.ConnectorMessage{
				Protocol: "ssh",
			},
		},
	}

	ctx := r.makeMemoryContext("thing", false, true)
	if ctx.user != "" {
		t.Fatalf("shared channel memory context user = %q, want empty", ctx.user)
	}
}
