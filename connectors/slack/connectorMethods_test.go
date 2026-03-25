package slack

import (
	"testing"
)

func TestFormatHelp(t *testing.T) {
	s := &slackConnector{}
	if got := s.FormatHelp("(alias) help <keyword> - find help"); got != "`(alias) help <keyword>` - find help" {
		t.Fatalf("FormatHelp() dash format = %q", got)
	}
	if got := s.FormatHelp("Usage: (alias) help <keyword>"); got != "Usage: `(alias) help <keyword>`" {
		t.Fatalf("FormatHelp() usage format = %q", got)
	}
}

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
