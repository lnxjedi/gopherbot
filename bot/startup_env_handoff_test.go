package bot

import (
	"os"
	"strings"
	"testing"
)

func preserveGopherEnvMaps(t *testing.T) {
	t.Helper()
	gopherEnvMutex.Lock()
	oldGopher := make(map[string]string, len(gopherEnv))
	for key, value := range gopherEnv {
		oldGopher[key] = value
	}
	oldStartup := make(map[string]string, len(startupEnv))
	for key, value := range startupEnv {
		oldStartup[key] = value
	}
	gopherEnv = make(map[string]string)
	startupEnv = make(map[string]string)
	gopherEnvMutex.Unlock()
	t.Cleanup(func() {
		gopherEnvMutex.Lock()
		gopherEnv = oldGopher
		startupEnv = oldStartup
		gopherEnvMutex.Unlock()
	})
}

func TestReadGopherEnvHandoffStoresInternalAndStartupEnv(t *testing.T) {
	preserveGopherEnvMaps(t)
	raw, err := encodeGopherEnvHandoff(map[string]string{
		"GOPHER_ENCRYPTION_KEY": "enc-secret",
		"GOPHER_DEPLOY_KEY":     "deploy-secret",
	})
	if err != nil {
		t.Fatalf("encodeGopherEnvHandoff(): %v", err)
	}

	if err := readGopherEnvHandoff(strings.NewReader(string(raw))); err != nil {
		t.Fatalf("readGopherEnvHandoff(): %v", err)
	}
	if got, ok := lookupEnv("GOPHER_ENCRYPTION_KEY"); !ok || got != "enc-secret" {
		t.Fatalf("GOPHER_ENCRYPTION_KEY = %q, present=%v; want handoff value", got, ok)
	}
	startup := copyStartupEnv()
	if got := startup["GOPHER_DEPLOY_KEY"]; got != "deploy-secret" {
		t.Fatalf("startup GOPHER_DEPLOY_KEY = %q, want handoff value", got)
	}
}

func TestEncodeGopherEnvHandoffRejectsNonGopherKey(t *testing.T) {
	if _, err := encodeGopherEnvHandoff(map[string]string{"PATH": "/bin"}); err == nil {
		t.Fatal("encodeGopherEnvHandoff accepted non-GOPHER key")
	}
}

func TestTakeReadEnvFromFD(t *testing.T) {
	t.Setenv(gopherEnvHandoffFDEnv, "7")

	fd, ok, err := takeReadEnvFromFD()
	if err != nil {
		t.Fatalf("takeReadEnvFromFD(): %v", err)
	}
	if !ok || fd != 7 {
		t.Fatalf("fd=%d ok=%v, want fd 7", fd, ok)
	}
	if got := os.Getenv(gopherEnvHandoffFDEnv); got != "" {
		t.Fatalf("%s still set to %q after take", gopherEnvHandoffFDEnv, got)
	}
}

func TestNewSelfCommandWithGopherEnvironmentUsesFDHandoff(t *testing.T) {
	preserveGopherEnvMaps(t)
	setStartupEnvValue("GOPHER_ENCRYPTION_KEY", "enc-secret")
	t.Setenv("GOPHER_ENCRYPTION_KEY", "direct-secret")
	t.Setenv("GOPHER_PROTOCOL", "ssh")

	cmd, cleanup, err := newSelfCommandWithGopherEnvironment("/bin/gopherbot", []string{"run"})
	if err != nil {
		t.Fatalf("newSelfCommandWithGopherEnvironment(): %v", err)
	}
	defer cleanup()
	if len(cmd.ExtraFiles) != 1 {
		t.Fatalf("ExtraFiles len = %d, want 1", len(cmd.ExtraFiles))
	}
	if got := strings.Join(cmd.Args, " "); got != "/bin/gopherbot run" {
		t.Fatalf("cmd args = %q, want no handoff flag", got)
	}
	joinedEnv := strings.Join(cmd.Env, "\n")
	if strings.Contains(joinedEnv, "GOPHER_ENCRYPTION_KEY=") || strings.Contains(joinedEnv, "GOPHER_PROTOCOL=") {
		t.Fatalf("cmd env leaked direct GOPHER vars: %s", joinedEnv)
	}
	if !strings.Contains(joinedEnv, gopherEnvHandoffFDEnv+"=3") {
		t.Fatalf("cmd env missing fd marker %s=3: %s", gopherEnvHandoffFDEnv, joinedEnv)
	}
}
