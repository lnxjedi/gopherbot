package bot

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func resetCryptKeyForTest(t *testing.T) {
	oldConfigPath := configPath
	oldInitialized := cryptKey.initialized
	oldInitializing := cryptKey.initializing
	oldKey := append([]byte(nil), cryptKey.key...)

	configPath = t.TempDir()
	cryptKey.Lock()
	cryptKey.key = nil
	cryptKey.initialized = false
	cryptKey.initializing = false
	cryptKey.Unlock()

	t.Cleanup(func() {
		configPath = oldConfigPath
		cryptKey.Lock()
		cryptKey.key = oldKey
		cryptKey.initialized = oldInitialized
		cryptKey.initializing = oldInitializing
		cryptKey.Unlock()
	})
}

func TestInitCryptFallsBackToSharedKeyFileWhenEnvironmentSpecificMissing(t *testing.T) {
	resetCryptKeyForTest(t)
	t.Setenv("GOPHER_ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("GOPHER_ENVIRONMENT", "development")

	if !initCrypt() {
		t.Fatalf("initCrypt returned false")
	}

	if _, err := os.Stat(filepath.Join(configPath, encryptedKeyFile)); err != nil {
		t.Fatalf("expected shared encrypted key file to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configPath, encryptedKeyFile+".development")); !os.IsNotExist(err) {
		t.Fatalf("did not expect environment-specific encrypted key file, got err=%v", err)
	}
}

func TestInitCryptUsesEnvironmentSpecificKeyFileWhenPresent(t *testing.T) {
	resetCryptKeyForTest(t)
	t.Setenv("GOPHER_ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("GOPHER_ENVIRONMENT", "development")

	sharedKey := []byte("0123456789abcdef0123456789abcdef")
	envKey := []byte("fedcba9876543210fedcba9876543210")
	ik := []byte("12345678901234567890123456789012")

	writeEncryptedKey := func(path string, key []byte) {
		t.Helper()
		encrypted, err := encrypt(key, ik)
		if err != nil {
			t.Fatalf("encrypt(%s): %v", path, err)
		}
		payload := base64.StdEncoding.EncodeToString(encrypted)
		if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeEncryptedKey(filepath.Join(configPath, encryptedKeyFile), sharedKey)
	writeEncryptedKey(filepath.Join(configPath, encryptedKeyFile+".development"), envKey)

	if !initCrypt() {
		t.Fatalf("initCrypt returned false")
	}

	cryptKey.RLock()
	got := append([]byte(nil), cryptKey.key...)
	cryptKey.RUnlock()
	if string(got) != string(envKey) {
		t.Fatalf("expected env-specific decrypted key %q, got %q", string(envKey), string(got))
	}
}
