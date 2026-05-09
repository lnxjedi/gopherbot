package bot

import "testing"

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
