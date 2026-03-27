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

func TestUnsupportedHiddenCommandMessage(t *testing.T) {
	if got := unsupportedHiddenCommandMessage(""); got == "" {
		t.Fatal("unsupportedHiddenCommandMessage(\"\") returned empty string")
	}
	if got := unsupportedHiddenCommandMessage("test"); got != "This command isn't supported with test because hidden commands are unavailable for this connector. Check with the robot administrator." {
		t.Fatalf("unsupportedHiddenCommandMessage(test) = %q", got)
	}
}

func TestDefaultHiddenCommandHint(t *testing.T) {
	if got := defaultHiddenCommandHint(""); got != "Hidden commands must be addressed to the robot." {
		t.Fatalf("defaultHiddenCommandHint(\"\") = %q", got)
	}
	if got := defaultHiddenCommandHint("Clu"); got != "Hidden commands must be addressed to Clu." {
		t.Fatalf("defaultHiddenCommandHint(Clu) = %q", got)
	}
}
