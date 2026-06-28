package bot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrivateEnvironmentPreservesProcessEnv(t *testing.T) {
	preserveGopherEnvMaps(t)
	const fileOnlyKey = "GOPHER_START_ENV_TEST_FILE_ONLY"
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("GOPHER_ENVIRONMENT=file\n"+fileOnlyKey+"=from-file\n"), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Setenv("GOPHER_ENVIRONMENT", "shell")
	origFileOnly, hadFileOnly := os.LookupEnv(fileOnlyKey)
	if err := os.Unsetenv(fileOnlyKey); err != nil {
		t.Fatalf("unset %s: %v", fileOnlyKey, err)
	}
	t.Cleanup(func() {
		if hadFileOnly {
			_ = os.Setenv(fileOnlyKey, origFileOnly)
		} else {
			_ = os.Unsetenv(fileOnlyKey)
		}
	})

	if err := loadPrivateEnvironment(envFile); err != nil {
		t.Fatalf("loadPrivateEnvironment(): %v", err)
	}

	if got := os.Getenv("GOPHER_ENVIRONMENT"); got != "shell" {
		t.Fatalf("GOPHER_ENVIRONMENT = %q, want process env value", got)
	}
	if got, ok := lookupEnv(fileOnlyKey); !ok || got != "from-file" {
		t.Fatalf("%s = %q, want file value", fileOnlyKey, got)
	}
	if got := os.Getenv(fileOnlyKey); got != "" {
		t.Fatalf("%s leaked into direct process environment as %q", fileOnlyKey, got)
	}
}
