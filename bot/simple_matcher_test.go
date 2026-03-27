package bot

import (
	"regexp"
	"testing"
)

func mustCompileSimpleMatcherRegexp(t *testing.T, spec string) *regexp.Regexp {
	t.Helper()
	regex, err := compileSimpleMatcher(spec)
	if err != nil {
		t.Fatalf("compileSimpleMatcher(%q): %v", spec, err)
	}
	return regexp.MustCompile(`^(?s:\s*` + regex + `\s*)$`)
}

func TestCompileSimpleMatcherLiteralAndWhitespace(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "hello world")
	for _, input := range []string{
		"hello world",
		"HELLO WORLD",
		" hello   world ",
		"hello-world",
		"hello---world",
	} {
		if !re.MatchString(input) {
			t.Fatalf("SimpleMatcher literal failed to match %q", input)
		}
	}
	for _, input := range []string{
		"hello",
		"world hello",
		"hello_world",
	} {
		if re.MatchString(input) {
			t.Fatalf("SimpleMatcher literal unexpectedly matched %q", input)
		}
	}
}

func TestCompileSimpleMatcherOptionalAndAlternatives(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "tell me [another] [knock-knock] joke")
	for _, input := range []string{
		"tell me joke",
		"tell me another joke",
		"tell me knock-knock joke",
		"tell me another knock knock joke",
		"tell-me-another-knock-knock-joke",
	} {
		if !re.MatchString(input) {
			t.Fatalf("optional/alternative matcher failed to match %q", input)
		}
	}
	if re.MatchString("tell me another") {
		t.Fatal("optional matcher should not match incomplete command")
	}
}

func TestCompileSimpleMatcherTypedCaptures(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "block [ticket|story] <story:slug> [<reason:rest>]")

	matches := re.FindStringSubmatch("block story train-123 because prod is broken")
	if len(matches) != 3 {
		t.Fatalf("len(matches) = %d, want 3 (%v)", len(matches), matches)
	}
	if matches[1] != "train-123" || matches[2] != "because prod is broken" {
		t.Fatalf("captures = %#v, want train-123 / because prod is broken", matches[1:])
	}

	matches = re.FindStringSubmatch("BLOCK ticket train-123")
	if len(matches) != 3 {
		t.Fatalf("len(matches) = %d, want 3 (%v)", len(matches), matches)
	}
	if matches[1] != "train-123" || matches[2] != "" {
		t.Fatalf("captures = %#v, want train-123 / empty optional reason", matches[1:])
	}
}

func TestCompileSimpleMatcherBuiltInTypes(t *testing.T) {
	tests := []struct {
		kind    string
		match   string
		noMatch string
	}{
		{"token", "show foo/bar", "show foo bar"},
		{"ident", "show slack-prod", "show 9slack"},
		{"slug", "show train-123*", "show train/123"},
		{"number", "show -42", "show 4.2"},
		{"decimal", "show -4.2", "show nope"},
		{"bool", "show on", "show maybe"},
		{"dnsname", "show api.example.com", "show bad_host"},
		{"email", "show ops@example.com", "show nope"},
		{"url", "show https://example.com/runbook", "show example.com/runbook"},
		{"base64", "show QUJDRA==", "show nope!"},
		{"duration", "show 5m30s", "show minutes"},
		{"cidr", "show 10.0.0.0/24", "show 10.0.0.0"},
		{"ip", "show 10.0.0.5", "show not-an-ip"},
	}

	for _, tc := range tests {
		re := mustCompileSimpleMatcherRegexp(t, "show <"+tc.kind+">")
		if !re.MatchString(tc.match) {
			t.Fatalf("%s matcher failed to match %q", tc.kind, tc.match)
		}
		if re.MatchString(tc.noMatch) {
			t.Fatalf("%s matcher unexpectedly matched %q", tc.kind, tc.noMatch)
		}
	}
}

func TestCompileSimpleMatcherRejectsUnknownType(t *testing.T) {
	if _, err := compileSimpleMatcher("show <mystery>"); err == nil {
		t.Fatal("expected unknown type error")
	}
}

func TestCompileInputMatcherRules(t *testing.T) {
	command := &InputMatcher{Command: "hello", SimpleMatcher: "hello world"}
	if err := compileInputMatcher(command, true); err != nil {
		t.Fatalf("compileInputMatcher(command): %v", err)
	}
	if command.re == nil {
		t.Fatal("compileInputMatcher(command) did not set compiled regexp")
	}

	if err := compileInputMatcher(&InputMatcher{Regex: "ping", SimpleMatcher: "ping"}, true); err == nil {
		t.Fatal("expected Regex + SimpleMatcher conflict")
	}

	if err := compileInputMatcher(&InputMatcher{SimpleMatcher: "ping"}, false); err == nil {
		t.Fatal("expected SimpleMatcher rejection outside Commands")
	}
}
