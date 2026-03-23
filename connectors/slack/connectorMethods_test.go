package slack

import (
	"strings"
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

func TestHiddenCommandHint(t *testing.T) {
	s := &slackConnector{}
	if got := s.FormatHiddenCommandExample("(alias) help ping"); got != "" {
		t.Fatalf("FormatHiddenCommandExample() = %q, want empty for Slack", got)
	}
	if got := s.HiddenCommandHint(); !strings.Contains(got, "slash command") {
		t.Fatalf("HiddenCommandHint() = %q, want slash-command guidance", got)
	}
}
