package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestHiddenMessageAddressedToRobot(t *testing.T) {
	tests := []struct {
		name       string
		botMessage bool
		cmdMode    string
		want       bool
	}{
		{name: "slack slash", botMessage: true, cmdMode: "alias", want: true},
		{name: "addressed by name", botMessage: false, cmdMode: "name", want: true},
		{name: "addressed by alias rejected", botMessage: false, cmdMode: "alias", want: false},
		{name: "direct mode rejected", botMessage: false, cmdMode: "direct", want: false},
	}

	for _, tt := range tests {
		if got := hiddenMessageAddressedToRobot(tt.botMessage, tt.cmdMode); got != tt.want {
			t.Fatalf("%s: hiddenMessageAddressedToRobot(%t,%q) = %t, want %t", tt.name, tt.botMessage, tt.cmdMode, got, tt.want)
		}
	}
}

func TestUnsupportedPrivateCommandMessage(t *testing.T) {
	if got := unsupportedPrivateCommandMessage(""); got == "" {
		t.Fatal("unsupportedPrivateCommandMessage(\"\") returned empty string")
	}
	if got := unsupportedPrivateCommandMessage("test"); got != "This command isn't supported with test because private command transport is unavailable for this connector. Check with the robot administrator." {
		t.Fatalf("unsupportedPrivateCommandMessage(test) = %q", got)
	}
}

func TestDefaultPrivateCommandHint(t *testing.T) {
	if got := defaultPrivateCommandHint(""); got != "Private commands must be addressed to the robot." {
		t.Fatalf("defaultPrivateCommandHint(\"\") = %q", got)
	}
	if got := defaultPrivateCommandHint("Clu"); got != "Private commands must be addressed to Clu." {
		t.Fatalf("defaultPrivateCommandHint(Clu) = %q", got)
	}
}

func TestPrivateContextSatisfiesChannels(t *testing.T) {
	task := &Task{name: "scoped", Channels: []string{"ops"}}
	plugin := &Plugin{Task: task, RestrictPrivateChannels: true}
	tests := []struct {
		name     string
		channel  string
		incoming *robot.ConnectorMessage
		want     bool
	}{
		{
			name:     "hidden allowed channel",
			channel:  "ops",
			incoming: &robot.ConnectorMessage{HiddenMessage: true},
			want:     true,
		},
		{
			name:     "hidden wrong channel",
			channel:  "general",
			incoming: &robot.ConnectorMessage{HiddenMessage: true},
			want:     false,
		},
		{
			name:     "direct message rejected",
			channel:  "",
			incoming: &robot.ConnectorMessage{DirectMessage: true},
			want:     false,
		},
	}
	for _, tt := range tests {
		w := &worker{Channel: tt.channel, Incoming: tt.incoming}
		if got := w.privateContextSatisfiesChannels(task, plugin); got != tt.want {
			t.Fatalf("%s: privateContextSatisfiesChannels() = %t, want %t", tt.name, got, tt.want)
		}
	}
}

func TestPrivateContextSatisfiesChannelsWhenNotRestricted(t *testing.T) {
	task := &Task{name: "scoped", Channels: []string{"ops"}}
	plugin := &Plugin{Task: task, AllowedPrivateCommands: []string{"show"}}
	w := &worker{Incoming: &robot.ConnectorMessage{DirectMessage: true}}
	if !w.privateContextSatisfiesChannels(task, plugin) {
		t.Fatal("unrestricted private command should not enforce plugin Channels")
	}
}
