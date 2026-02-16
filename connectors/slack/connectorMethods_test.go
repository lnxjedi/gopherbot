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

func TestDefaultHelpIncludesCommands(t *testing.T) {
	s := &slackConnector{}
	lines := s.DefaultHelp()
	found := false
	for _, line := range lines {
		if strings.Contains(line, "commands") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("DefaultHelp() did not include commands line: %#v", lines)
	}
}
