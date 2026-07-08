package bot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptAndSyntaxCommandsRunBeforeFullInit(t *testing.T) {
	for _, command := range []string{"script", "syntax"} {
		if !cliCommandRunsBeforeInit(command) {
			t.Fatalf("%s command should run before full robot initialization", command)
		}
	}
}

func TestProcessCLIHelpScriptAndSyntax(t *testing.T) {
	for _, tc := range []struct {
		command string
		needles []string
	}{
		{
			command: "script",
			needles: []string{
				"Usage: gopherbot script [options] <script> [--] <command> [args...]",
				"conf/default-fixture.yaml",
				"-fixture <path>",
				"-new-fixture <path>",
				"-no-interactive",
			},
		},
		{
			command: "syntax",
			needles: []string{
				"Usage: gopherbot syntax [options] <script> [script...]",
				"-language <lua|js|gsh|go>",
				"-json, -j",
			},
		},
	} {
		output := captureStdout(t, func() {
			code := processCLI("help", []string{tc.command})
			if code != 0 {
				t.Fatalf("processCLI(help %s) = %d, want 0", tc.command, code)
			}
		})
		for _, needle := range tc.needles {
			if !strings.Contains(output, needle) {
				t.Fatalf("processCLI(help %s) missing %q in output:\n%s", tc.command, needle, output)
			}
		}
	}
}

func TestProcessCLISyntaxJSONReportsMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	validLua := filepath.Join(dir, "valid.lua")
	invalidJS := filepath.Join(dir, "invalid.js")
	if err := os.WriteFile(validLua, []byte("return 0\n"), 0600); err != nil {
		t.Fatalf("write valid lua: %v", err)
	}
	if err := os.WriteFile(invalidJS, []byte("function () {\n"), 0600); err != nil {
		t.Fatalf("write invalid js: %v", err)
	}

	output := captureStdout(t, func() {
		code := processCLI("syntax", []string{"-json", validLua, invalidJS})
		if code != 1 {
			t.Fatalf("processCLI(syntax -json) = %d, want 1", code)
		}
	})
	var report cliScriptSyntaxReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("syntax JSON unmarshal: %v\n%s", err, output)
	}
	if report.Status != "error" || len(report.Files) != 2 {
		t.Fatalf("syntax report = %#v, want error with two files", report)
	}
	if !report.Files[0].OK || report.Files[0].Language != "lua" {
		t.Fatalf("valid lua report = %#v, want OK lua", report.Files[0])
	}
	if report.Files[1].OK || report.Files[1].Language != "js" || len(report.Files[1].Diagnostics) == 0 {
		t.Fatalf("invalid js report = %#v, want diagnostic", report.Files[1])
	}
}

func TestParseCLIScriptInvocationArgsWithBoundary(t *testing.T) {
	inv, err := parseCLIScriptInvocationArgs([]string{"plugins/foo.lua", "--", "console", "-spot", "qa"}, cliScriptOptions{})
	if err != nil {
		t.Fatalf("parseCLIScriptInvocationArgs() error = %v", err)
	}
	if inv.ScriptPath != "plugins/foo.lua" {
		t.Fatalf("ScriptPath = %q, want plugins/foo.lua", inv.ScriptPath)
	}
	if got := strings.Join(inv.Args, "|"); got != "console|-spot|qa" {
		t.Fatalf("Args = %q, want console|-spot|qa", got)
	}
}

func TestMergeCLIScriptPluginArgsFromFixture(t *testing.T) {
	command, args := mergeCLIScriptArgs("plugin", nil, cliScriptFixture{
		Command: "prompt",
		Args:    []string{"cat"},
	})
	if command != "prompt" {
		t.Fatalf("command = %q, want prompt", command)
	}
	if got := strings.Join(args, "|"); got != "cat" {
		t.Fatalf("args = %q, want cat", got)
	}
}

func TestLoadCLIScriptYAMLFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.yaml")
	if err := os.WriteFile(path, []byte(`
message:
  user: bob
  channel: ops
parameters:
  CAT_COLOR: tuxedo
config:
  Openings:
    - YAML hello
prompts:
  replies:
    - Felix
memory:
  long_term:
    cat_profile:
      name: Felix
users:
  bob:
    fullName: Bob Example
    internalID: U456
`), 0600); err != nil {
		t.Fatalf("write yaml fixture: %v", err)
	}
	fixture, err := loadCLIScriptFixture(path)
	if err != nil {
		t.Fatalf("loadCLIScriptFixture() error = %v", err)
	}
	if fixture.Message.User != "bob" || fixture.Message.Channel != "ops" {
		t.Fatalf("fixture message = %#v, want bob in ops", fixture.Message)
	}
	if got := fixture.Parameters["CAT_COLOR"]; got != "tuxedo" {
		t.Fatalf("CAT_COLOR = %q, want tuxedo", got)
	}
	var cfg struct {
		Openings []string
	}
	if err := json.Unmarshal(fixture.Config, &cfg); err != nil {
		t.Fatalf("unmarshal config raw JSON: %v", err)
	}
	if len(cfg.Openings) != 1 || cfg.Openings[0] != "YAML hello" {
		t.Fatalf("config openings = %#v, want YAML hello", cfg.Openings)
	}
	if got := string(fixture.Memory.LongTerm["cat_profile"]); !strings.Contains(got, "Felix") {
		t.Fatalf("long-term memory raw JSON = %s, want Felix", got)
	}
}

func TestProcessCLIScriptNewFixtureCopiesInstalledDefault(t *testing.T) {
	oldInstallPath := installPath
	installRoot := t.TempDir()
	installPath = installRoot
	t.Cleanup(func() {
		installPath = oldInstallPath
	})

	src := filepath.Join(installRoot, "conf", "default-fixture.yaml")
	if err := os.MkdirAll(filepath.Dir(src), 0700); err != nil {
		t.Fatalf("mkdir conf: %v", err)
	}
	const fixtureTemplate = "# default fixture\nmessage:\n  user: alice\n"
	if err := os.WriteFile(src, []byte(fixtureTemplate), 0600); err != nil {
		t.Fatalf("write installed default fixture: %v", err)
	}
	dest := filepath.Join(t.TempDir(), "my-fixture.yaml")

	output := captureStdout(t, func() {
		code := processCLI("script", []string{"--new-fixture", dest})
		if code != 0 {
			t.Fatalf("processCLI(script --new-fixture) = %d, want 0", code)
		}
	})
	if !strings.Contains(output, "Created fixture "+dest) {
		t.Fatalf("new-fixture output = %q, want destination", output)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read copied fixture: %v", err)
	}
	if string(data) != fixtureTemplate {
		t.Fatalf("copied fixture = %q, want template", string(data))
	}

	stderr := captureStderr(t, func() {
		output = captureStdout(t, func() {
			code := processCLI("script", []string{"--new-fixture", dest})
			if code != 1 {
				t.Fatalf("processCLI(script --new-fixture existing) = %d, want 1", code)
			}
		})
	})
	if !strings.Contains(output, "already exists") && !strings.Contains(stderr, "already exists") {
		t.Fatalf("existing fixture output missing error; stdout=%q stderr=%q", output, stderr)
	}
}
