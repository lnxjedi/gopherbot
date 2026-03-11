package bot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCryptIgnoresGopherEnvironmentForKeyFile(t *testing.T) {
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
