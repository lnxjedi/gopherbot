package bot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Close(stdout writer): %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Copy(stdout): %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close(stdout reader): %v", err)
	}
	return buf.String()
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Close(stderr writer): %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Copy(stderr): %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close(stderr reader): %v", err)
	}
	return buf.String()
}

func TestPrintCLIUsageIncludesHelpDiscovery(t *testing.T) {
	output := captureStdout(t, func() {
		printCLIUsage()
	})
	for _, needle := range []string{
		"Usage: gopherbot [options] [command [command options] [command args]]",
		"help [command]",
		"gopherbot help <command>",
		"gopherbot <command> -h",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("printCLIUsage() missing %q in output:\n%s", needle, output)
		}
	}
}

func TestProcessCLIHelpEncryptShowsCommandDetails(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("help", []string{"encrypt"})
		if code != 0 {
			t.Fatalf("processCLI(help encrypt) = %d, want 0", code)
		}
	})
	for _, needle := range []string{
		"Usage: gopherbot encrypt [options] <string>",
		"-f, -file <path|->",
		"-b, -binary",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("processCLI(help encrypt) missing %q in output:\n%s", needle, output)
		}
	}
}

func TestProcessCLIHelpUUIDShowsCommandDetails(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("help", []string{"uuid"})
		if code != 0 {
			t.Fatalf("processCLI(help uuid) = %d, want 0", code)
		}
	})
	for _, needle := range []string{
		"Usage: gopherbot uuid",
		"Generates a random UUID",
		"encrypted value suitable for custom/conf/variables/<environment>.yaml",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("processCLI(help uuid) missing %q in output:\n%s", needle, output)
		}
	}
}

func TestCLIUUIDRunsBeforeFullInit(t *testing.T) {
	if !cliCommandRunsBeforeInit("uuid") {
		t.Fatal("uuid command should run before full robot initialization")
	}
}

func TestUserCLICommandsRunBeforeFullInit(t *testing.T) {
	for _, command := range []string{
		"check",
		"delete",
		"decrypt",
		"dump",
		"encrypt",
		"fetch",
		"flush-brain",
		"genkey",
		"gentotp",
		"help",
		"init",
		"list",
		"match",
		"store",
		"uuid",
		"validate",
		"version",
	} {
		if !cliCommandRunsBeforeInit(command) {
			t.Fatalf("%s command should run before full robot initialization", command)
		}
	}
}

func TestProcessCLICheckReportsCaptures(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("check", []string{"spot (type:rails|devops) up [<branch:token>]", "spot-devops-up"})
		if code != 0 {
			t.Fatalf("processCLI(check) = %d, want 0", code)
		}
	})
	if got, want := strings.TrimSpace(output), `MATCH ["devops", ""]`; got != want {
		t.Fatalf("processCLI(check) output = %q, want %q", got, want)
	}
}

func TestProcessCLICheckReportsNoCapturesAsEmptyList(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("check", []string{"spot devops up", "spot-devops-up"})
		if code != 0 {
			t.Fatalf("processCLI(check no captures) = %d, want 0", code)
		}
	})
	if got, want := strings.TrimSpace(output), `MATCH []`; got != want {
		t.Fatalf("processCLI(check no captures) output = %q, want %q", got, want)
	}
}

func TestProcessCLICheckReportsSyntaxDiagnostic(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("check", []string{"set loglevel {to} (level:trace|debug|info|warn|error)", "set-loglevel-fine"})
		if code != 1 {
			t.Fatalf("processCLI(check syntax) = %d, want 1", code)
		}
	})
	if !strings.Contains(output, `SYNTAX Invalid value: "fine" for: "level"; valid values: trace, debug, info, warn, error.`) {
		t.Fatalf("processCLI(check syntax) output missing diagnostic:\n%s", output)
	}
}

func TestProcessCLICheckJSONReportsCaptures(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("check", []string{"-json", "spot (type:rails|devops) up [<branch:token>]", "spot-devops-up"})
		if code != 0 {
			t.Fatalf("processCLI(check -json) = %d, want 0", code)
		}
	})
	var report cliCommandMatchReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("check JSON unmarshal: %v\n%s", err, output)
	}
	if report.Status != "match" || len(report.Matches) != 1 {
		t.Fatalf("check JSON report = %#v, want one match", report)
	}
	if got := report.Matches[0].Args; len(got) != 2 || got[0] != "devops" || got[1] != "" {
		t.Fatalf("check JSON args = %#v, want [devops \"\"]", got)
	}
}

func TestMatchConfiguredCommandReportsExactAndAmbiguousMatches(t *testing.T) {
	first := testCLIPluginWithCommands(t, "remote-devel", []InputMatcher{{
		Command:       "spot-up",
		SimpleMatcher: "spot (type:rails|devops) up [<branch:token>]",
	}})
	second := testCLIPluginWithCommands(t, "alias-devel", []InputMatcher{{
		Command:       "spot-up",
		SimpleMatcher: "spot (type:rails|devops) up [<branch:token>]",
	}})
	report := matchConfiguredCommand("spot-devops-up", &taskList{t: []interface{}{nil, first, second}})
	if report.Status != "match" || !report.Ambiguous || len(report.Matches) != 2 {
		t.Fatalf("configured match report = %#v, want two ambiguous matches", report)
	}
	for _, match := range report.Matches {
		if len(match.Args) != 2 || match.Args[0] != "devops" || match.Args[1] != "" {
			t.Fatalf("configured match args = %#v, want [devops \"\"]", match.Args)
		}
	}
}

func TestMatchConfiguredCommandHyphenatedCommandDisambiguatesArguments(t *testing.T) {
	generic := testCLIPluginWithCommands(t, "remote-devel", []InputMatcher{{
		Command:       "spot-up",
		SimpleMatcher: "spot (type:rails|devops) up [<branch:token>]",
	}})
	specific := testCLIPluginWithCommands(t, "remote-devel-dev", []InputMatcher{{
		Command:       "spot-up",
		SimpleMatcher: "spot (type:rails|devops) up dev",
	}})
	tasks := &taskList{t: []interface{}{nil, generic, specific}}

	report := matchConfiguredCommand("spot devops up dev", tasks)
	if report.Status != "match" || !report.Ambiguous || len(report.Matches) != 2 {
		t.Fatalf("spaced command report = %#v, want two ambiguous matches", report)
	}

	report = matchConfiguredCommand("spot-devops-up dev", tasks)
	if report.Status != "match" || report.Ambiguous || len(report.Matches) != 1 {
		t.Fatalf("hyphenated command with branch report = %#v, want one generic match", report)
	}
	if got := report.Matches[0]; got.Plugin != "remote-devel" || got.Command != "spot-up" || len(got.Args) != 2 || got.Args[0] != "devops" || got.Args[1] != "dev" {
		t.Fatalf("hyphenated command with branch match = %#v, want remote-devel/spot-up [devops dev]", got)
	}

	report = matchConfiguredCommand("spot-devops-up-dev", tasks)
	if report.Status != "match" || report.Ambiguous || len(report.Matches) != 1 {
		t.Fatalf("hyphenated specific command report = %#v, want one specific match", report)
	}
	if got := report.Matches[0]; got.Plugin != "remote-devel-dev" || got.Command != "spot-up" || len(got.Args) != 1 || got.Args[0] != "devops" {
		t.Fatalf("hyphenated specific command match = %#v, want remote-devel-dev/spot-up [devops]", got)
	}

	report = matchConfiguredCommand("spot devops-up-dev", tasks)
	if report.Status != "no_match" {
		t.Fatalf("mixed separator command report = %#v, want no_match", report)
	}
}

func TestMatchConfiguredCommandReportsSyntaxDiagnostics(t *testing.T) {
	plugin := testCLIPluginWithCommands(t, "builtin-logging", []InputMatcher{{
		Command:       "loglevel",
		SimpleMatcher: "set loglevel {to} (level:trace|debug|info|warn|error)",
	}})
	report := matchConfiguredCommand("set-loglevel-fine", &taskList{t: []interface{}{nil, plugin}})
	if report.Status != "syntax" || len(report.SyntaxDiagnostics) != 1 {
		t.Fatalf("configured syntax report = %#v, want one syntax diagnostic", report)
	}
	if got := report.SyntaxDiagnostics[0].Diagnostic; !strings.Contains(got, `Invalid value: "fine" for: "level"`) {
		t.Fatalf("syntax diagnostic = %q, want invalid level", got)
	}
}

func TestProcessCLIMatchInteractivePromptsUntilEOF(t *testing.T) {
	var prompts bytes.Buffer
	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			code := processCLIMatchInteractive(strings.NewReader("spot-devops-up\n\nunknown\n"), &prompts, false, func(input string) cliCommandMatchReport {
				report := cliCommandMatchReport{
					Input:           input,
					Status:          "no_match",
					RedactedSecrets: true,
				}
				if input == "spot-devops-up" {
					report.Status = "match"
					report.Matches = append(report.Matches, cliCommandMatch{
						Plugin:  "remote-devel",
						Command: "spot-up",
						Args:    []string{"devops", ""},
					})
				}
				return report
			})
			if code != 0 {
				t.Fatalf("processCLIMatchInteractive() = %d, want 0", code)
			}
		})
	})
	if got, want := prompts.String(), "Command?: Command?: Command?: Command?: "; got != want {
		t.Fatalf("prompts = %q, want %q", got, want)
	}
	if strings.Count(stderr, "redacted secret placeholders") != 1 {
		t.Fatalf("stderr should contain one redaction notice, got:\n%s", stderr)
	}
	if !strings.Contains(stdout, `MATCH remote-devel/spot-up ["devops", ""]`) {
		t.Fatalf("stdout missing match:\n%s", stdout)
	}
	if strings.Count(stdout, "NO MATCH") != 1 {
		t.Fatalf("stdout should contain one no-match result, got:\n%s", stdout)
	}
}

func TestProcessCLIMatchInteractiveRejectsCommandArgs(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLIMatchCommand([]string{"spot-devops-up"}, false, true)
		if code != 2 {
			t.Fatalf("processCLIMatchCommand(interactive with args) = %d, want 2", code)
		}
	})
	if !strings.Contains(output, "match -interactive does not accept command text arguments") {
		t.Fatalf("output missing interactive argument error:\n%s", output)
	}
}

func testCLIPluginWithCommands(t *testing.T, name string, commands []InputMatcher) *Plugin {
	t.Helper()
	plugin := &Plugin{
		Task: &Task{name: name},
	}
	plugin.Commands = append([]InputMatcher(nil), commands...)
	for i := range plugin.Commands {
		if err := compileInputMatcher(&plugin.Commands[i], true); err != nil {
			t.Fatalf("compileInputMatcher(%s/%s): %v", name, plugin.Commands[i].Command, err)
		}
	}
	return plugin
}

func TestProcessCLIHelpBrainMemoryCommandsShowCacheSemantics(t *testing.T) {
	for command, needles := range map[string][]string{
		"fetch": {
			"By default, fetch reads the local cache only",
			"-validate-cloud",
			"-cloud",
		},
		"store": {
			"Usage: gopherbot store <key> [file]",
			"flushes cloud sync",
		},
		"delete": {
			"Usage: gopherbot delete <key>",
			"delete tombstone to cloud",
		},
		"flush-brain": {
			"Usage: gopherbot flush-brain",
			"queued local brain cache writes",
		},
		"list": {
			"Usage: gopherbot list [options]",
			"-cloud",
		},
		"restore-brain": {
			"Usage: gopherbot restore-brain [-v2] [options]",
			"Defaults to v3 output",
			"-v2",
		},
	} {
		output := captureStdout(t, func() {
			code := processCLI("help", []string{command})
			if code != 0 {
				t.Fatalf("processCLI(help %s) = %d, want 0", command, code)
			}
		})
		for _, needle := range needles {
			if !strings.Contains(output, needle) {
				t.Fatalf("processCLI(help %s) missing %q in output:\n%s", command, needle, output)
			}
		}
	}
}

func TestProcessCLIHelpDumpShowsRedactedSecretDefault(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("help", []string{"dump"})
		if code != 0 {
			t.Fatalf("processCLI(help dump) = %d, want 0", code)
		}
	})
	for _, needle := range []string{
		"Usage: gopherbot dump [options] <installed|configured> <path>",
		"Secret template values are redacted by default",
		"-unredacted-secrets",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("processCLI(help dump) missing %q in output:\n%s", needle, output)
		}
	}
}

func TestCLIDumpRedactsSecretTemplatesByDefault(t *testing.T) {
	resetConfigVariableTestState(t)
	configPath = t.TempDir()
	writeCLIDumpSecretFixture(t, configPath)

	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			cliDump("configured", "plugins/secret.yaml", false)
		})
	})
	if !strings.Contains(stdout, "token: "+redactedTemplateSecret) {
		t.Fatalf("dump stdout missing redacted token:\n%s", stdout)
	}
	if strings.Contains(stdout, "secret-value") {
		t.Fatalf("dump stdout exposed secret:\n%s", stdout)
	}
	if !strings.Contains(stderr, "secret template values redacted") {
		t.Fatalf("dump stderr missing redaction notice:\n%s", stderr)
	}
}

func TestCLIDumpUnredactedSecretsPrintsDecryptedTemplateValues(t *testing.T) {
	resetConfigVariableTestState(t)
	configPath = t.TempDir()
	writeCLIDumpSecretFixture(t, configPath)

	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			cliDump("configured", "plugins/secret.yaml", true)
		})
	})
	if !strings.Contains(stdout, "token: secret-value") {
		t.Fatalf("dump stdout missing decrypted token:\n%s", stdout)
	}
	if strings.Contains(stdout, redactedTemplateSecret) {
		t.Fatalf("dump stdout still redacted:\n%s", stdout)
	}
	if strings.Contains(stderr, "secret template values redacted") {
		t.Fatalf("dump stderr had redaction notice despite unredacted mode:\n%s", stderr)
	}
}

func writeCLIDumpSecretFixture(t *testing.T, root string) {
	t.Helper()
	varDir := filepath.Join(root, "conf", "variables")
	pluginDir := filepath.Join(root, "conf", "plugins")
	if err := os.MkdirAll(varDir, 0700); err != nil {
		t.Fatalf("MkdirAll variables: %v", err)
	}
	if err := os.MkdirAll(pluginDir, 0700); err != nil {
		t.Fatalf("MkdirAll plugins: %v", err)
	}
	if err := os.WriteFile(filepath.Join(varDir, "common.yaml"), []byte(`
Secrets:
  API_TOKEN: "`+encryptedConfigSecretForTest(t, "secret-value")+`"
`), 0600); err != nil {
		t.Fatalf("write variables: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "secret.yaml"), []byte(`token: {{ secret "API_TOKEN" }}
`), 0600); err != nil {
		t.Fatalf("write plugin config: %v", err)
	}
}

func TestRestoreBrainDefaultsToV3WithV2Override(t *testing.T) {
	if got := (brainRestoreOptions{}).effectiveRemoteFormat(); got != "v3" {
		t.Fatalf("default restore format = %q, want v3", got)
	}
	if got := (brainRestoreOptions{v2: true}).effectiveRemoteFormat(); got != "v2" {
		t.Fatalf("-v2 restore format = %q, want v2", got)
	}
}

func TestGenerateEncryptedUUID(t *testing.T) {
	oldKey := cryptKey.key
	oldInitialized := cryptKey.initialized
	testKey := []byte("0123456789abcdef0123456789abcdef")
	cryptKey.Lock()
	cryptKey.key = testKey
	cryptKey.initialized = true
	cryptKey.Unlock()
	t.Cleanup(func() {
		cryptKey.Lock()
		cryptKey.key = oldKey
		cryptKey.initialized = oldInitialized
		cryptKey.Unlock()
	})

	plain, encrypted, err := generateEncryptedUUID()
	if err != nil {
		t.Fatalf("generateEncryptedUUID() error = %v", err)
	}
	if _, err := uuid.Parse(plain); err != nil {
		t.Fatalf("generated plaintext is not a UUID: %v", err)
	}
	ct, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("encrypted UUID is not base64: %v", err)
	}
	decrypted, err := decrypt(ct, testKey)
	if err != nil {
		t.Fatalf("decrypt(encrypted UUID) error = %v", err)
	}
	if string(decrypted) != plain {
		t.Fatalf("decrypted UUID = %q, want %q", decrypted, plain)
	}
}

func TestEncryptPlaintextBase64(t *testing.T) {
	oldKey := cryptKey.key
	oldInitialized := cryptKey.initialized
	testKey := []byte("0123456789abcdef0123456789abcdef")
	cryptKey.Lock()
	cryptKey.key = testKey
	cryptKey.initialized = true
	cryptKey.Unlock()
	t.Cleanup(func() {
		cryptKey.Lock()
		cryptKey.key = oldKey
		cryptKey.initialized = oldInitialized
		cryptKey.Unlock()
	})

	const plaintext = "hunter2"
	encrypted, err := encryptPlaintextBase64(plaintext)
	if err != nil {
		t.Fatalf("encryptPlaintextBase64() error = %v", err)
	}
	if strings.Contains(encrypted, plaintext) {
		t.Fatalf("encrypted value contains plaintext: %q", encrypted)
	}
	ct, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("encrypted value is not base64: %v", err)
	}
	decrypted, err := decrypt(ct, testKey)
	if err != nil {
		t.Fatalf("decrypt(encrypted secret) error = %v", err)
	}
	if string(decrypted) != plaintext {
		t.Fatalf("decrypted secret = %q, want %q", decrypted, plaintext)
	}
}

func TestProcessCLIEncryptHelpFlagShowsHelp(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("encrypt", []string{"-h"})
		if code != 0 {
			t.Fatalf("processCLI(encrypt -h) = %d, want 0", code)
		}
	})
	if !strings.Contains(output, "Usage: gopherbot encrypt [options] <string>") {
		t.Fatalf("encrypt -h output missing usage:\n%s", output)
	}
}

func TestShouldShowCLICommandHelpForValidateFlag(t *testing.T) {
	if !shouldShowCLICommandHelp("validate", []string{"-h"}) {
		t.Fatal("expected validate -h to be recognized as help")
	}
	if shouldShowCLICommandHelp("validate", []string{"/tmp/robot"}) {
		t.Fatal("did not expect validate path arg to be recognized as help")
	}
}

func TestProcessCLIUnknownCommandShowsError(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("wat", nil)
		if code != 2 {
			t.Fatalf("processCLI(unknown) = %d, want 2", code)
		}
	})
	if !strings.Contains(output, `Error: unknown command "wat"`) {
		t.Fatalf("unknown command output missing error:\n%s", output)
	}
}
