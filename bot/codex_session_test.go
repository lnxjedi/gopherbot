package bot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestCodexSanitizePathPart(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "Alice", want: "alice"},
		{input: "dev/team@prod", want: "dev_team_prod"},
		{input: "___HELLO---", want: "hello"},
		{input: " ./. ", want: ""},
	}
	for _, tc := range tests {
		got := codexSanitizePathPart(tc.input)
		if got != tc.want {
			t.Fatalf("codexSanitizePathPart(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCodexPathWithin(t *testing.T) {
	base := "/tmp/base"
	if !codexPathWithin(base, "/tmp/base") {
		t.Fatalf("expected exact base path to be within base")
	}
	if !codexPathWithin(base, "/tmp/base/child") {
		t.Fatalf("expected child path to be within base")
	}
	if codexPathWithin(base, "/tmp/other") {
		t.Fatalf("expected unrelated path to be outside base")
	}
	if codexPathWithin(base, "/tmp/base/../other") {
		t.Fatalf("expected traversed outside path to be outside base")
	}
}

func TestCodexSessionKeyFromRobot(t *testing.T) {
	r := Robot{
		Message: &robot.Message{
			Channel: "general",
			Protocol: robot.SSH,
			Incoming: &robot.ConnectorMessage{
				Protocol: "ssh",
				ThreadID: "0005",
			},
		},
	}
	key, ok := codexSessionKeyFromRobot(r)
	if !ok {
		t.Fatalf("expected session key to be derived")
	}
	if key.Protocol != "ssh" || key.Channel != "general" || key.ThreadID != "0005" {
		t.Fatalf("unexpected key: %+v", key)
	}
}

func TestCodexResolveWorkspaceDir(t *testing.T) {
	oldHome := homePath
	oldPrivSep := privSep
	home := t.TempDir()
	homePath = home
	privSep = false
	t.Cleanup(func() {
		homePath = oldHome
		privSep = oldPrivSep
	})

	target := filepath.Join(home, "work", "nested")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("creating target dir: %v", err)
	}

	abs, label, err := codexResolveWorkspaceDir("work/nested")
	if err != nil {
		t.Fatalf("codexResolveWorkspaceDir returned error: %v", err)
	}
	if abs != target {
		t.Fatalf("resolved path = %q, want %q", abs, target)
	}
	if label != filepath.Clean("work/nested") {
		t.Fatalf("label = %q, want %q", label, filepath.Clean("work/nested"))
	}
}

func TestCodexResolveWorkspaceDirRejectsEscape(t *testing.T) {
	oldHome := homePath
	oldPrivSep := privSep
	home := t.TempDir()
	homePath = home
	privSep = false
	t.Cleanup(func() {
		homePath = oldHome
		privSep = oldPrivSep
	})

	if _, _, err := codexResolveWorkspaceDir("../outside"); err == nil {
		t.Fatalf("expected escape path to be rejected")
	}
}
