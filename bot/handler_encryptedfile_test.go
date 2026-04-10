package bot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerReadEncryptedFileUsesConfigRoot(t *testing.T) {
	oldConfigFull := configFull
	oldInstallPath := installPath
	oldCrypt := cryptKey
	configFull = t.TempDir()
	installPath = t.TempDir()
	cryptKey.key = []byte("12345678901234567890123456789012")
	cryptKey.initialized = true
	cryptKey.initializing = false
	t.Cleanup(func() {
		configFull = oldConfigFull
		installPath = oldInstallPath
		cryptKey = oldCrypt
	})

	plaintext := []byte("{\"type\":\"service_account\"}\n")
	ciphertext, err := encrypt(plaintext, cryptKey.key)
	if err != nil {
		t.Fatalf("encrypt() error: %v", err)
	}
	if err := WriteBase64File(filepath.Join(configFull, "gopherbot-key.json.enc"), &ciphertext); err != nil {
		t.Fatalf("WriteBase64File(): %v", err)
	}

	got, err := (handler{}).ReadEncryptedFile("gopherbot-key.json.enc")
	if err != nil {
		t.Fatalf("ReadEncryptedFile() error: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("ReadEncryptedFile() = %q, want %q", string(got), string(plaintext))
	}
}

func TestHandlerReadEncryptedFileFallsBackToInstallRoot(t *testing.T) {
	oldConfigFull := configFull
	oldInstallPath := installPath
	oldCrypt := cryptKey
	configFull = t.TempDir()
	installPath = t.TempDir()
	cryptKey.key = []byte("12345678901234567890123456789012")
	cryptKey.initialized = true
	cryptKey.initializing = false
	t.Cleanup(func() {
		configFull = oldConfigFull
		installPath = oldInstallPath
		cryptKey = oldCrypt
	})

	plaintext := []byte("install-root-secret")
	ciphertext, err := encrypt(plaintext, cryptKey.key)
	if err != nil {
		t.Fatalf("encrypt() error: %v", err)
	}
	if err := WriteBase64File(filepath.Join(installPath, "shared.enc"), &ciphertext); err != nil {
		t.Fatalf("WriteBase64File(): %v", err)
	}

	got, err := (handler{}).ReadEncryptedFile("shared.enc")
	if err != nil {
		t.Fatalf("ReadEncryptedFile() error: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("ReadEncryptedFile() = %q, want %q", string(got), string(plaintext))
	}
}

func TestHandlerReadEncryptedFileRejectsTraversalOutsideRoots(t *testing.T) {
	oldConfigFull := configFull
	oldInstallPath := installPath
	oldCrypt := cryptKey
	configFull = t.TempDir()
	installPath = t.TempDir()
	cryptKey.key = []byte("12345678901234567890123456789012")
	cryptKey.initialized = true
	cryptKey.initializing = false
	t.Cleanup(func() {
		configFull = oldConfigFull
		installPath = oldInstallPath
		cryptKey = oldCrypt
	})

	outsideDir := t.TempDir()
	target := filepath.Join(outsideDir, "secret.enc")
	if err := os.WriteFile(target, []byte("not-used"), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	if _, err := (handler{}).ReadEncryptedFile(target); err == nil {
		t.Fatal("ReadEncryptedFile() succeeded for file outside allowed roots")
	}
}
