package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func newAvailabilityTestWorker(user, channel string, direct bool) *worker {
	return &worker{
		User:     user,
		Channel:  channel,
		cfg:      &configuration{},
		Incoming: &robot.ConnectorMessage{DirectMessage: direct},
	}
}

func TestCommandLocationHintSingleChannel(t *testing.T) {
	w := newAvailabilityTestWorker("alice", "general", false)
	task := &Task{name: "rubydemo", Channels: []string{"random"}}
	plugin := &Plugin{Task: task}

	hint, ok := w.commandLocationHint(task, plugin, "ruby")
	if !ok {
		t.Fatalf("commandLocationHint() = not ok, want hint")
	}
	got := hint.format(w.Channel, w.Incoming.DirectMessage)
	want := "rubydemo/ruby not available in #general, try #random"
	if got != want {
		t.Fatalf("commandLocationHint().format() = %q, want %q", got, want)
	}
}

func TestCommandLocationHintMultipleChannels(t *testing.T) {
	w := newAvailabilityTestWorker("alice", "deadzone", false)
	task := &Task{name: "echo", Channels: []string{"general", "random"}}
	plugin := &Plugin{Task: task}

	hint, ok := w.commandLocationHint(task, plugin, "echo")
	if !ok {
		t.Fatalf("commandLocationHint() = not ok, want hint")
	}
	got := hint.format(w.Channel, w.Incoming.DirectMessage)
	want := "echo/echo not available in #deadzone, try one of: #general, #random"
	if got != want {
		t.Fatalf("commandLocationHint().format() = %q, want %q", got, want)
	}
}

func TestCommandLocationHintAnyRegularChannelFromDM(t *testing.T) {
	w := newAvailabilityTestWorker("alice", "", true)
	task := &Task{name: "bashdemo", AllChannels: true}
	plugin := &Plugin{Task: task}

	hint, ok := w.commandLocationHint(task, plugin, "hear")
	if !ok {
		t.Fatalf("commandLocationHint() = not ok, want hint")
	}
	got := hint.format(w.Channel, w.Incoming.DirectMessage)
	want := "bashdemo/hear not available in direct messages, try it in any regular channel"
	if got != want {
		t.Fatalf("commandLocationHint().format() = %q, want %q", got, want)
	}
}

func TestCommandLocationHintPrivateOnly(t *testing.T) {
	w := newAvailabilityTestWorker("alice", "general", false)
	task := &Task{name: "log", AllChannels: true}
	plugin := &Plugin{Task: task, RequiredPrivateCommands: []string{"show"}}

	hint, ok := w.commandLocationHint(task, plugin, "show")
	if !ok {
		t.Fatalf("commandLocationHint() = not ok, want hint")
	}
	got := hint.format(w.Channel, w.Incoming.DirectMessage)
	want := "log/show not available in #general, try a private context"
	if got != want {
		t.Fatalf("commandLocationHint().format() = %q, want %q", got, want)
	}
}

func TestCommandLocationHintSuppressesUnauthorizedUser(t *testing.T) {
	w := newAvailabilityTestWorker("alice", "general", false)
	task := &Task{name: "ping", Channels: []string{"random"}, Users: []string{"bob"}}
	plugin := &Plugin{Task: task}

	if _, ok := w.commandLocationHint(task, plugin, "ping"); ok {
		t.Fatalf("commandLocationHint() ok = true, want false for unauthorized user")
	}
}

func TestCommandLocationHintSuppressesUnknownAuthorizer(t *testing.T) {
	w := newAvailabilityTestWorker("alice", "general", false)
	task := &Task{
		name:        "opaqueauth",
		Channels:    []string{"random"},
		Authorizer:  "opaque",
		AuthRequire: "Helpdesk",
	}
	plugin := &Plugin{
		Task:               task,
		AuthorizedCommands: []string{"echo"},
	}

	if _, ok := w.commandLocationHint(task, plugin, "echo"); ok {
		t.Fatalf("commandLocationHint() ok = true, want false when authorizer visibility is unknown")
	}
}
