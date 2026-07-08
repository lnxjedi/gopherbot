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
				"-fixture <path>",
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
