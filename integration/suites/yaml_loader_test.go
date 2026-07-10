package suites

import (
	"os"
	"testing"
)

func TestYAMLInputUserNamesResolveToConnectorIDs(t *testing.T) {
	c, err := yamlCaseToCase(yamlCase{
		Input: yamlMessage{
			User: Alice,
			Text: "ping",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Input.User != AliceID {
		t.Fatalf("input user = %q, want %q", c.Input.User, AliceID)
	}
}

func TestWithDevelopmentEnvironmentRestoresPreviousValue(t *testing.T) {
	previous, existed := os.LookupEnv("GOPHER_ENVIRONMENT")
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv("GOPHER_ENVIRONMENT", previous)
		} else {
			_ = os.Unsetenv("GOPHER_ENVIRONMENT")
		}
	})

	if err := os.Setenv("GOPHER_ENVIRONMENT", "production"); err != nil {
		t.Fatal(err)
	}
	cleanup, err := withDevelopmentEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOPHER_ENVIRONMENT"); got != "development" {
		t.Fatalf("GOPHER_ENVIRONMENT = %q, want development", got)
	}
	cleanup()
	if got := os.Getenv("GOPHER_ENVIRONMENT"); got != "production" {
		t.Fatalf("restored GOPHER_ENVIRONMENT = %q, want production", got)
	}
}
