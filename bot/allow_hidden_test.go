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
