package bot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInstallPathFollowsExecutableSymlink(t *testing.T) {
	realDir := t.TempDir()
	linkDir := t.TempDir()
	realExe := filepath.Join(realDir, "gopherbot")
	if err := os.WriteFile(realExe, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	linkExe := filepath.Join(linkDir, "gopherbot")
	if err := os.Symlink(realExe, linkExe); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	got, err := resolveInstallPath(linkExe)
	if err != nil {
		t.Fatal(err)
	}
	if got != realDir {
		want, err := filepath.EvalSymlinks(realDir)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("resolveInstallPath(%q) = %q, want %q", linkExe, got, want)
		}
	}
}

func TestResolveInstallPathUsesExecutableDirWhenNotSymlink(t *testing.T) {
	realDir := t.TempDir()
	realExe := filepath.Join(realDir, "gopherbot")
	if err := os.WriteFile(realExe, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := resolveInstallPath(realExe)
	if err != nil {
		t.Fatal(err)
	}
	if got != realDir {
		want, err := filepath.EvalSymlinks(realDir)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("resolveInstallPath(%q) = %q, want %q", realExe, got, want)
		}
	}
}
