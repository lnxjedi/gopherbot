package bot

import (
	"strings"
	"testing"
)

func TestSanitizedChildEnvironmentStripsSensitiveKeys(t *testing.T) {
	t.Setenv("GOPHER_ENCRYPTION_KEY", "enc-secret")
	t.Setenv("GOPHER_DEPLOY_KEY", "deploy-secret")
	t.Setenv("GOPHER_HOST_KEYS", "host-secret")
	t.Setenv("GOPHER_PROTOCOL", "ssh")
	t.Setenv(gopherEnvHandoffFDEnv, "3")

	env := sanitizedChildEnvironment("GOPHER_PROTOCOL=test", "TEST_EXTRA=value")
	joined := strings.Join(env, "\n")

	for _, key := range []string{"GOPHER_ENCRYPTION_KEY=", "GOPHER_DEPLOY_KEY=", "GOPHER_HOST_KEYS=", "GOPHER_PROTOCOL=ssh", gopherEnvHandoffFDEnv + "="} {
		if strings.Contains(joined, key) {
			t.Fatalf("inherited GOPHER key %s leaked into child environment", key)
		}
	}
	if !strings.Contains(joined, "GOPHER_PROTOCOL=test") {
		t.Fatalf("expected explicit GOPHER extra env var to be present")
	}
	if !strings.Contains(joined, "TEST_EXTRA=value") {
		t.Fatalf("expected explicit extra env var to be present")
	}
}
