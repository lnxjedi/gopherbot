package bot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptedKeyFilePathsProduction(t *testing.T) {
	origConfigPath := configPath
	origDeployEnv := deployEnvironment
	t.Cleanup(func() {
		configPath = origConfigPath
		deployEnvironment = origDeployEnv
	})

	configPath = "custom"
	deployEnvironment = "production"

	preferred, fallback := encryptedKeyFilePaths()
	if preferred != filepath.Join("custom", encryptedKeyFile) {
		t.Fatalf("preferred = %q", preferred)
	}
	if fallback != "" {
		t.Fatalf("fallback = %q, want empty", fallback)
	}
}

func TestEncryptedKeyFilePathsNonProduction(t *testing.T) {
	origConfigPath := configPath
	origDeployEnv := deployEnvironment
	t.Cleanup(func() {
		configPath = origConfigPath
		deployEnvironment = origDeployEnv
	})

	configPath = "custom"
	deployEnvironment = "development"

	preferred, fallback := encryptedKeyFilePaths()
	if preferred != filepath.Join("custom", encryptedKeyFile+".development") {
		t.Fatalf("preferred = %q", preferred)
	}
	if fallback != filepath.Join("custom", encryptedKeyFile) {
		t.Fatalf("fallback = %q", fallback)
	}
}

func TestResolveEncryptedKeyFileUsesEnvSpecificWhenPresent(t *testing.T) {
	tmpDir := t.TempDir()
	origConfigPath := configPath
	origDeployEnv := deployEnvironment
	t.Cleanup(func() {
		configPath = origConfigPath
		deployEnvironment = origDeployEnv
	})

	configPath = tmpDir
	deployEnvironment = "development"

	envPath := filepath.Join(tmpDir, encryptedKeyFile+".development")
	if err := os.WriteFile(envPath, []byte("env"), 0600); err != nil {
		t.Fatalf("WriteFile(envPath): %v", err)
	}
	basePath := filepath.Join(tmpDir, encryptedKeyFile)
	if err := os.WriteFile(basePath, []byte("base"), 0600); err != nil {
		t.Fatalf("WriteFile(basePath): %v", err)
	}

	loadPath, createPath, usedFallback, err := resolveEncryptedKeyFile()
	if err != nil {
		t.Fatalf("resolveEncryptedKeyFile(): %v", err)
	}
	if loadPath != envPath {
		t.Fatalf("loadPath = %q, want %q", loadPath, envPath)
	}
	if createPath != envPath {
		t.Fatalf("createPath = %q, want %q", createPath, envPath)
	}
	if usedFallback {
		t.Fatalf("usedFallback = true, want false")
	}
}

func TestResolveEncryptedKeyFileFallsBackToBaseWhenEnvSpecificMissing(t *testing.T) {
	tmpDir := t.TempDir()
	origConfigPath := configPath
	origDeployEnv := deployEnvironment
	t.Cleanup(func() {
		configPath = origConfigPath
		deployEnvironment = origDeployEnv
	})

	configPath = tmpDir
	deployEnvironment = "development"

	basePath := filepath.Join(tmpDir, encryptedKeyFile)
	if err := os.WriteFile(basePath, []byte("base"), 0600); err != nil {
		t.Fatalf("WriteFile(basePath): %v", err)
	}

	loadPath, createPath, usedFallback, err := resolveEncryptedKeyFile()
	if err != nil {
		t.Fatalf("resolveEncryptedKeyFile(): %v", err)
	}
	if loadPath != basePath {
		t.Fatalf("loadPath = %q, want %q", loadPath, basePath)
	}
	if createPath != basePath {
		t.Fatalf("createPath = %q, want %q", createPath, basePath)
	}
	if !usedFallback {
		t.Fatalf("usedFallback = false, want true")
	}
}

func TestResolveEncryptedKeyFileCreatesBaseWhenNoCandidatesExist(t *testing.T) {
	tmpDir := t.TempDir()
	origConfigPath := configPath
	origDeployEnv := deployEnvironment
	t.Cleanup(func() {
		configPath = origConfigPath
		deployEnvironment = origDeployEnv
	})

	configPath = tmpDir
	deployEnvironment = "development"

	loadPath, createPath, usedFallback, err := resolveEncryptedKeyFile()
	if err != nil {
		t.Fatalf("resolveEncryptedKeyFile(): %v", err)
	}
	if loadPath != "" {
		t.Fatalf("loadPath = %q, want empty", loadPath)
	}
	wantCreate := filepath.Join(tmpDir, encryptedKeyFile)
	if createPath != wantCreate {
		t.Fatalf("createPath = %q, want %q", createPath, wantCreate)
	}
	if !usedFallback {
		t.Fatalf("usedFallback = false, want true")
	}
}
