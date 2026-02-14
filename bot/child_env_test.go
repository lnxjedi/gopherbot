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

	env := sanitizedChildEnvironment("TEST_EXTRA=value")
	joined := strings.Join(env, "\n")

	for _, key := range []string{"GOPHER_ENCRYPTION_KEY=", "GOPHER_DEPLOY_KEY=", "GOPHER_HOST_KEYS="} {
		if strings.Contains(joined, key) {
			t.Fatalf("sensitive key %s leaked into child environment", key)
		}
	}
	if !strings.Contains(joined, "GOPHER_PROTOCOL=ssh") {
		t.Fatalf("expected non-sensitive GOPHER key to remain in child environment")
	}
	if !strings.Contains(joined, "TEST_EXTRA=value") {
		t.Fatalf("expected explicit extra env var to be present")
	}
}
