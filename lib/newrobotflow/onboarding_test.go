package newrobotflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteInitialEnvOnlyKeepsEncryptionKeyBootstrapState(t *testing.T) {
	tempDir := t.TempDir()
	prevDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(prevDir); chdirErr != nil {
			t.Fatalf("restoring cwd failed: %v", chdirErr)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) failed: %v", tempDir, err)
	}

	original := strings.Join([]string{
		"KEEP_ME=yes",
		"GOPHER_CUSTOM_REPOSITORY=git@github.com:example/robot.git",
		"GOPHER_DEPLOY_KEY=oldkey",
		"GOPHER_BOTNAME=clu",
		"GOPHER_ENVIRONMENT=development",
		"",
	}, "\n")
	if err := os.WriteFile(".env", []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile(.env) failed: %v", err)
	}

	const key = "12345678901234567890123456789012"
	if err := writeInitialEnv(key); err != nil {
		t.Fatalf("writeInitialEnv() failed: %v", err)
	}

	body, err := os.ReadFile(".env")
	if err != nil {
		t.Fatalf("ReadFile(.env) failed: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "GOPHER_ENCRYPTION_KEY="+key) {
		t.Fatalf(".env missing encryption key: %q", got)
	}
	for _, unwanted := range []string{
		"GOPHER_CUSTOM_REPOSITORY=",
		"GOPHER_DEPLOY_KEY=",
		"GOPHER_BOTNAME=",
		"GOPHER_ENVIRONMENT=development",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf(".env still contains %q: %q", unwanted, got)
		}
	}
	if !strings.Contains(got, "KEEP_ME=yes") {
		t.Fatalf(".env dropped unrelated content: %q", got)
	}
	if !strings.Contains(got, "# GOPHER_ENVIRONMENT=production") {
		t.Fatalf(".env missing environment guidance comment: %q", got)
	}
}

func TestEnsureSSHProtocolUserKeyReplacesTemplateBlock(t *testing.T) {
	tempDir := t.TempDir()
	sshConfigPath := filepath.Join(tempDir, "ssh.yaml")
	original := strings.Join([]string{
		"ProtocolConfig:",
		"  ListenHost: localhost",
		"  DefaultChannel: general",
		"  UserKeys: []",
		"  # - UserName: johndoe",
		"  #   PublicKeys:",
		"  #   - \"ssh-ed25519 AAAA...firstkey\"",
		"",
	}, "\n")
	if err := os.WriteFile(sshConfigPath, []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", sshConfigPath, err)
	}

	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyExample samantha@example"
	if err := ensureSSHProtocolUserKey(sshConfigPath, "samantha", key); err != nil {
		t.Fatalf("ensureSSHProtocolUserKey() failed: %v", err)
	}

	body, err := os.ReadFile(sshConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", sshConfigPath, err)
	}
	got := string(body)
	if strings.Contains(got, "UserKeys: []") {
		t.Fatalf("ssh config still contains placeholder UserKeys block: %q", got)
	}
	expected := strings.Join([]string{
		"  UserKeys:",
		"  - UserName: \"samantha\"",
		"    PublicKeys:",
		"    - \"" + key + "\"",
	}, "\n")
	if !strings.Contains(got, expected) {
		t.Fatalf("ssh config missing rewritten UserKeys block:\nwant substring:\n%s\n\ngot:\n%s", expected, got)
	}
	if !strings.Contains(got, "# - UserName: johndoe") {
		t.Fatalf("ssh config lost surrounding comments: %q", got)
	}
}

func TestFindSessionForJoinMatchesCanonicalUser(t *testing.T) {
	state := setupStateFile{
		Sessions: map[string]setupSession{
			"alice": {
				Status:        statusCompleted,
				Stage:         stageRepoReady,
				CanonicalUser: "samantha",
			},
		},
	}
	key, session, found := findSessionForJoin(state, "samantha")
	if !found {
		t.Fatal("findSessionForJoin() did not find canonical user match")
	}
	if key != "alice" {
		t.Fatalf("findSessionForJoin() key = %q, want alice", key)
	}
	if session.CanonicalUser != "samantha" {
		t.Fatalf("findSessionForJoin() canonical user = %q", session.CanonicalUser)
	}
}
