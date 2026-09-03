package help

import (
	"strings"
	"testing"
)

func TestBuildHelpReplyFormatsStarAliasAsInlineCode(t *testing.T) {
	got := buildHelpReply("bishop", "*", "admin@example.com", "")
	if !strings.Contains(got, "**Help**") {
		t.Fatalf("buildHelpReply() missing markdown heading: %q", got)
	}
	if !strings.Contains(got, "Hi, I'm Bishop, a staff robot. I see you've asked for help.") {
		t.Fatalf("buildHelpReply() missing friendly intro: %q", got)
	}
	if !strings.Contains(got, "alias `*`") {
		t.Fatalf("buildHelpReply() missing literal alias display: %q", got)
	}
	if !strings.Contains(got, "`*help ping`") {
		t.Fatalf("buildHelpReply() missing literal star alias command: %q", got)
	}
	if strings.Contains(got, "*help ping*") {
		t.Fatalf("buildHelpReply() rendered alias command as emphasis: %q", got)
	}
}

func TestPreferredCommandExampleFallsBackToBotName(t *testing.T) {
	got := preferredCommandExample("bishop", "", "help ping")
	if got != "`bishop, help ping`" {
		t.Fatalf("preferredCommandExample() = %q, want %q", got, "`bishop, help ping`")
	}
}

func TestBuildHelpReplyExplainsSummaryAndExactHelp(t *testing.T) {
	got := buildHelpReply("bishop", ";", "", "")
	if !strings.Contains(got, "Most browse and search results are terse, one-line summaries.") {
		t.Fatalf("buildHelpReply() missing terse-help guidance: %q", got)
	}
	if !strings.Contains(got, "For full usage, options, examples, and availability for one command, use `;help <plugin>/<command>`.") {
		t.Fatalf("buildHelpReply() missing exact-help guidance: %q", got)
	}
}

func TestDisplayBotNameCapitalizesFirstLetter(t *testing.T) {
	if got := displayBotName("bishop"); got != "Bishop" {
		t.Fatalf("displayBotName() = %q, want %q", got, "Bishop")
	}
}
