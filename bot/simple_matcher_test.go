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

	reSingle := mustCompileSimpleMatcherRegexp(t, "reload")
	if !reSingle.MatchString("reload") {
		t.Fatalf("SimpleMatcher literal failed to match single word %q", "reload")
	}
	if !reSingle.MatchString("  reload  ") {
		t.Fatalf("SimpleMatcher literal failed to match single word with spaces")
	}
}

func TestCompileSimpleMatcherOptionalAndAlternatives(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "tell me {another} {knock-knock} joke")
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
	re := mustCompileSimpleMatcherRegexp(t, "block /ticket|story/ <story:slug> [<reason:rest>]")

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

func TestCompileSimpleMatcherRejectsBareAlternation(t *testing.T) {
	if _, err := compileSimpleMatcher("get|take <thing:rest>"); err == nil {
		t.Fatal("expected bare alternation error")
	}
}

func TestCompileSimpleMatcherRequiredCapturingChoice(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "set log level {to} (level:trace|debug|info|warn|error)")

	matches := re.FindStringSubmatch("set log-level to debug")
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2 (%v)", len(matches), matches)
	}
	if matches[1] != "debug" {
		t.Fatalf("captures = %#v, want debug", matches[1:])
	}

	matches = re.FindStringSubmatch("set log level warn")
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2 (%v)", len(matches), matches)
	}
	if matches[1] != "warn" {
		t.Fatalf("captures = %#v, want warn", matches[1:])
	}

	matches = re.FindStringSubmatch("set log level trace")
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2 (%v)", len(matches), matches)
	}
	if matches[1] != "trace" {
		t.Fatalf("captures = %#v, want trace", matches[1:])
	}
}

func TestCompileSimpleMatcherEmptyLabelChoiceAllowsColonValues(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "set target (:foo:bar|baz|frotz)")

	for _, tc := range []struct {
		input string
		want  string
	}{
		{"set target foo:bar", "foo:bar"},
		{"set target baz", "baz"},
		{"set target frotz", "frotz"},
	} {
		matches := re.FindStringSubmatch(tc.input)
		if len(matches) != 2 {
			t.Fatalf("len(matches) = %d for %q, want 2 (%v)", len(matches), tc.input, matches)
		}
		if matches[1] != tc.want {
			t.Fatalf("captures for %q = %#v, want %q", tc.input, matches[1:], tc.want)
		}
	}
}

func TestCompileSimpleMatcherRejectsUnlabelledCapturingChoices(t *testing.T) {
	for _, spec := range []string{
		"set log level {to} (trace|debug|info)",
		"set feature <name:ident> [disabled]",
	} {
		if _, err := compileSimpleMatcher(spec); err == nil {
			t.Fatalf("compileSimpleMatcher(%q) succeeded, want labelled choice error", spec)
		}
	}
}

func TestCompileSimpleMatcherRequiredNonCapturingSynonyms(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "/remove|delete/ <user:token> from {the} <group:rest> group")

	for _, input := range []string{
		"remove alice from ops group",
		"delete alice from the ops group",
	} {
		matches := re.FindStringSubmatch(input)
		if len(matches) != 3 {
			t.Fatalf("len(matches) = %d for %q, want 3 (%v)", len(matches), input, matches)
		}
		if matches[1] != "alice" || matches[2] != "ops" {
			t.Fatalf("captures for %q = %#v, want alice / ops", input, matches[1:])
		}
	}
}

func TestCompileSimpleMatcherRequiredNonCapturingPhraseSynonyms(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "/pick up|take|grab/ <item:rest>")

	for _, input := range []string{
		"pick up wrench",
		"pick-up wrench",
		"take wrench",
		"grab wrench",
	} {
		matches := re.FindStringSubmatch(input)
		if len(matches) != 2 {
			t.Fatalf("len(matches) = %d for %q, want 2 (%v)", len(matches), input, matches)
		}
		if matches[1] != "wrench" {
			t.Fatalf("captures for %q = %#v, want wrench", input, matches[1:])
		}
	}
}

func TestCompileSimpleMatcherRejectsNestedCapturesInGroups(t *testing.T) {
	tests := []string{
		"show {<name:ident>}",
		"show /<name:ident>|all/",
		"show (prefix <name:ident>)",
	}
	for _, spec := range tests {
		if _, err := compileSimpleMatcher(spec); err == nil {
			t.Fatalf("compileSimpleMatcher(%q) succeeded, want error", spec)
		}
	}
}

func TestCompileSimpleMatcherShippedConfigArgPositions(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		input   string
		capture []string
	}{
		{
			name:    "logging level captures selected level only",
			spec:    "set log level {to} (level:trace|debug|info|warn|error)",
			input:   "set log-level to debug",
			capture: []string{"debug"},
		},
		{
			name:    "logging page captures page only",
			spec:    "show /log|logs/ [page <page:number>]",
			input:   "show logs page 2",
			capture: []string{"2"},
		},
		{
			name:    "logging page omitted captures empty page",
			spec:    "show /log|logs/ [page <page:number>]",
			input:   "show log",
			capture: []string{""},
		},
		{
			name:    "logging lines ignores noise",
			spec:    "set log lines {to} <lines:number>",
			input:   "set log lines to 3",
			capture: []string{"3"},
		},
		{
			name:    "groups add ignores article",
			spec:    "add <user:token> to {the} [<group:rest>] group",
			input:   "add alice to the Helpdesk group",
			capture: []string{"alice", "Helpdesk"},
		},
		{
			name:    "groups remove ignores synonym and article",
			spec:    "/remove|delete/ <user:token> from {the} [<group:rest>] group",
			input:   "delete alice from the Helpdesk group",
			capture: []string{"alice", "Helpdesk"},
		},
		{
			name:    "admin branch ignores synonym",
			spec:    "/switch|change/ branch <branch:token>",
			input:   "change branch feature/test",
			capture: []string{"feature/test"},
		},
		{
			name:    "admin git info synonyms do not capture",
			spec:    "/git info|branch info|show branch/",
			input:   "show branch",
			capture: []string{},
		},
		{
			name:    "ping thread ignores optional verb",
			spec:    "{new|start|create} thread [<topic:rest>]",
			input:   "new thread db migration",
			capture: []string{"db migration"},
		},
		{
			name:    "admin ps optional mode",
			spec:    "ps [<mode:token>]",
			input:   "ps -v",
			capture: []string{"-v"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			re := mustCompileSimpleMatcherRegexp(t, tc.spec)
			matches := re.FindStringSubmatch(tc.input)
			if matches == nil {
				t.Fatalf("SimpleMatcher %q did not match %q", tc.spec, tc.input)
			}
			if len(matches[1:]) != len(tc.capture) {
				t.Fatalf("captures = %#v, want %#v", matches[1:], tc.capture)
			}
			for i, want := range tc.capture {
				if matches[i+1] != want {
					t.Fatalf("captures = %#v, want %#v", matches[1:], tc.capture)
				}
			}
		})
	}
}

func TestCapturingOptional(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "set log-level <level:ident> [:disabled]")

	matches := re.FindStringSubmatch("set log-level debug disabled")
	if len(matches) != 3 {
		t.Fatalf("len(matches) = %d, want 3 (%v)", len(matches), matches)
	}
	if matches[1] != "debug" || matches[2] != "disabled" {
		t.Fatalf("captures = %#v, want debug / disabled", matches[1:])
	}

	matches = re.FindStringSubmatch("set log-level debug")
	if len(matches) != 3 {
		t.Fatalf("len(matches) = %d, want 3 (%v)", len(matches), matches)
	}
	if matches[1] != "debug" || matches[2] != "" {
		t.Fatalf("captures = %#v, want debug / empty", matches[1:])
	}
}

func TestNoiseWordsGroup(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "enable {the} feature <name:ident>")

	for _, input := range []string{
		"enable the feature foo",
		"enable feature foo",
	} {
		matches := re.FindStringSubmatch(input)
		if len(matches) != 2 {
			t.Fatalf("len(matches) = %d for %q, want 2", len(matches), input)
		}
		if matches[1] != "foo" {
			t.Fatalf("captures = %#v, want foo", matches[1:])
		}
	}
}

func TestNoiseWordsGroupDoesNotCapture(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "set log lines {to} <lines:number>")

	matches := re.FindStringSubmatch("set log lines to 3")
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2 (%v)", len(matches), matches)
	}
	if matches[1] != "3" {
		t.Fatalf("captures = %#v, want 3", matches[1:])
	}
}

func TestMixedGroupsGroupCount(t *testing.T) {
	re := mustCompileSimpleMatcherRegexp(t, "[:verbose] show {me} <name:ident>")

	matches := re.FindStringSubmatch("verbose show me foo")
	if len(matches) != 3 {
		t.Fatalf("len(matches) = %d, want 3", len(matches))
	}
	if matches[1] != "verbose" || matches[2] != "foo" {
		t.Fatalf("captures = %#v, want verbose / foo", matches[1:])
	}

	matches = re.FindStringSubmatch("show foo")
	if len(matches) != 3 {
		t.Fatalf("len(matches) = %d, want 3", len(matches))
	}
	if matches[1] != "" || matches[2] != "foo" {
		t.Fatalf("captures = %#v, want empty / foo", matches[1:])
	}
}

func TestCapturingOptionalWithSlots(t *testing.T) {
	// If [...] contains a slot, the [...] itself should be non-capturing to avoid double-captures.
	// This means we expect only ONE capture group (from the slot itself).
	re := mustCompileSimpleMatcherRegexp(t, "show [<name:ident>]")

	matches := re.FindStringSubmatch("show debug")
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2", len(matches))
	}
	if matches[1] != "debug" {
		t.Fatalf("captures = %#v, want debug", matches[1:])
	}

	matches = re.FindStringSubmatch("show")
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2", len(matches))
	}
	if matches[1] != "" {
		t.Fatalf("captures = %#v, want empty", matches[1:])
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

func mustCompileSimpleInputMatcher(t *testing.T, spec string) InputMatcher {
	t.Helper()
	matcher := InputMatcher{Command: "test", SimpleMatcher: spec}
	if err := compileInputMatcher(&matcher, true); err != nil {
		t.Fatalf("compileInputMatcher(%q): %v", spec, err)
	}
	return matcher
}

func assertInputMatchResult(t *testing.T, result inputMatchResult, wantKind inputMatchKind, wantArgs []string, wantDiagnostic string) {
	t.Helper()
	if result.kind != wantKind {
		t.Fatalf("kind = %v, want %v (result: %#v)", result.kind, wantKind, result)
	}
	if len(result.args) != len(wantArgs) {
		t.Fatalf("args = %#v, want %#v", result.args, wantArgs)
	}
	for i, want := range wantArgs {
		if result.args[i] != want {
			t.Fatalf("args = %#v, want %#v", result.args, wantArgs)
		}
	}
	if result.diagnostic != wantDiagnostic {
		t.Fatalf("diagnostic = %q, want %q", result.diagnostic, wantDiagnostic)
	}
}

func TestSimpleMatcherInputMatchExactLabelledChoice(t *testing.T) {
	matcher := mustCompileSimpleInputMatcher(t, "set loglevel {to} (level:trace|debug|info|warn|error)")

	result := matcher.matchInput("set-loglevel to debug")

	assertInputMatchResult(t, result, inputExactMatch, []string{"debug"}, "")
}

func TestSimpleMatcherInputMatchSyntaxDiagnosticForLabelledChoice(t *testing.T) {
	matcher := mustCompileSimpleInputMatcher(t, "set loglevel {to} (level:trace|debug|info|warn|error)")

	result := matcher.matchInput("set loglevel to fine")

	assertInputMatchResult(t, result, inputSyntaxMatch, nil, "Invalid value 'fine' for 'level'; valid values: trace, debug, info, warn, error.")
}

func TestSimpleMatcherInputMatchSkeletonMismatchIsNoMatch(t *testing.T) {
	matcher := mustCompileSimpleInputMatcher(t, "set loglevel {to} (level:trace|debug|info|warn|error)")

	for _, input := range []string{
		"set logging to fine",
		"set loglevel to fine now",
		"loglevel to fine",
	} {
		result := matcher.matchInput(input)
		assertInputMatchResult(t, result, inputNoMatch, nil, "")
	}
}

func TestSimpleMatcherInputMatchSyntaxDiagnosticForTypedCapture(t *testing.T) {
	matcher := mustCompileSimpleInputMatcher(t, "deploy siding <siding:ident>")

	result := matcher.matchInput("deploy siding 9round")

	assertInputMatchResult(t, result, inputSyntaxMatch, nil, "Invalid value '9round' for 'siding'; expected an identifier starting with a letter, followed by letters, numbers, '_' or '-'.")
}

func TestSimpleMatcherInputMatchOptionalTypedCaptureDiagnostics(t *testing.T) {
	matcher := mustCompileSimpleInputMatcher(t, "show /log|logs/ [page <page:number>]")

	result := matcher.matchInput("show logs page two")
	assertInputMatchResult(t, result, inputSyntaxMatch, nil, "Invalid value 'two' for 'page'; expected an integer.")

	result = matcher.matchInput("show logs two")
	assertInputMatchResult(t, result, inputNoMatch, nil, "")

	result = matcher.matchInput("show logs")
	assertInputMatchResult(t, result, inputExactMatch, []string{""}, "")
}
