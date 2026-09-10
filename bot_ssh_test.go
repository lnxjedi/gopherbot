package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBotSSHKnownHostsUsesNamedHostKeyWithLegacyFallback(t *testing.T) {
	script, err := filepath.Abs("bot-ssh")
	if err != nil {
		t.Fatalf("resolve bot-ssh: %v", err)
	}
	workdir := t.TempDir()
	custom := filepath.Join(workdir, "custom")
	if err := os.Mkdir(custom, 0700); err != nil {
		t.Fatalf("create custom directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, ".ssh-connect"), []byte("BOT_SSH_PORT=127.0.0.1:4221\nBOT_SERVER_PUBKEY='ssh-ed25519 runtime'\n"), 0600); err != nil {
		t.Fatalf("write .ssh-connect: %v", err)
	}
	legacyPath := filepath.Join(custom, "robot-ssh.pub")
	if err := os.WriteFile(legacyPath, []byte("ssh-ed25519 legacy old-comment\n"), 0600); err != nil {
		t.Fatalf("write legacy host key: %v", err)
	}
	newPath := filepath.Join(custom, "ssh-host-key.pub")
	if err := os.WriteFile(newPath, []byte("ssh-ed25519 current new-comment\n"), 0600); err != nil {
		t.Fatalf("write named host key: %v", err)
	}

	if got := runBotSSHKnownHosts(t, script, workdir); got != "[127.0.0.1]:4221 ssh-ed25519 current" {
		t.Fatalf("new host-key output = %q", got)
	}
	if err := os.Remove(newPath); err != nil {
		t.Fatalf("remove named host key: %v", err)
	}
	if got := runBotSSHKnownHosts(t, script, workdir); got != "[127.0.0.1]:4221 ssh-ed25519 legacy" {
		t.Fatalf("legacy host-key output = %q", got)
	}
}

func runBotSSHKnownHosts(t *testing.T, script, workdir string) string {
	t.Helper()
	cmd := exec.Command(script, "-k")
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bot-ssh -k: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}
