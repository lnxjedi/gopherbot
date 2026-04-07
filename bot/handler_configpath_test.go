package bot

import "testing"

func TestHandlerGetConfigPathUsesConfigFull(t *testing.T) {
	oldConfigPath := configPath
	oldConfigFull := configFull
	oldInstallPath := installPath
	configPath = "custom"
	configFull = "/tmp/example-config"
	installPath = "/tmp/example-install"
	t.Cleanup(func() {
		configPath = oldConfigPath
		configFull = oldConfigFull
		installPath = oldInstallPath
	})

	got := (handler{}).GetConfigPath()
	if got != configFull {
		t.Fatalf("GetConfigPath() = %q, want %q", got, configFull)
	}
}

func TestHandlerGetConfigPathFallsBackToInstallPath(t *testing.T) {
	oldConfigFull := configFull
	oldInstallPath := installPath
	configFull = ""
	installPath = "/tmp/example-install"
	t.Cleanup(func() {
		configFull = oldConfigFull
		installPath = oldInstallPath
	})

	got := (handler{}).GetConfigPath()
	if got != installPath {
		t.Fatalf("GetConfigPath() = %q, want %q", got, installPath)
	}
}
