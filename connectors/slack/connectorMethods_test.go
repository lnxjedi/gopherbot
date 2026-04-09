package slack

import (
	"testing"
)

func TestDefaultHelpUsesEngineDefault(t *testing.T) {
	s := &slackConnector{}
	lines := s.DefaultHelp()
	if len(lines) != 0 {
		t.Fatalf("DefaultHelp() = %#v, want nil/empty to defer to engine defaults", lines)
	}
}

func TestFormatHiddenCommand(t *testing.T) {
	s := &slackConnector{slashCommand: "clu"}
	if got := s.FormatHiddenCommand("help ping"); got != "/clu help ping" {
		t.Fatalf("FormatHiddenCommand() = %q", got)
	}
}
