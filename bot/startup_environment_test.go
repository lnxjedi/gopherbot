package bot

import (
	"strings"
	"testing"
)

func TestValidateStartupEnvironmentMatrix(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		environment string
		wantError   string
	}{
		{name: "demo without environment", mode: "demo"},
		{name: "demo with environment", mode: "demo", environment: "development"},
		{name: "cli defers configuration requirement", mode: "cli"},
		{name: "bootstrap requires environment", mode: "bootstrap", wantError: "required when GOPHER_CUSTOM_REPOSITORY is configured"},
		{name: "production requires environment", mode: "production", wantError: "required when GOPHER_CUSTOM_REPOSITORY is configured"},
		{name: "test development requires environment", mode: "test-dev", wantError: "required for startup mode"},
		{name: "configured environment must be valid", mode: "production", environment: "../production", wantError: "single path segment"},
		{name: "configured environment is accepted", mode: "production", environment: "production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preserveGopherEnvMaps(t)
			unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
			if tt.environment != "" {
				setGopherEnvValue("GOPHER_ENVIRONMENT", tt.environment)
			}
			err := validateStartupEnvironment(tt.mode)
			if tt.wantError == "" && err != nil {
				t.Fatalf("validateStartupEnvironment(%q) error = %v, want nil", tt.mode, err)
			}
			if tt.wantError != "" && (err == nil || !strings.Contains(err.Error(), tt.wantError)) {
				t.Fatalf("validateStartupEnvironment(%q) error = %v, want substring %q", tt.mode, err, tt.wantError)
			}
		})
	}
}

func TestDetectStartupModeTreatsBlankRepositoryAsUnset(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")
	setGopherEnvValue("GOPHER_ENVIRONMENT", "development")
	setGopherEnvValue("GOPHER_CUSTOM_REPOSITORY", "   ")

	if got := detectStartupMode(); got != "demo" {
		t.Fatalf("detectStartupMode() = %q, want demo for blank repository", got)
	}
}

func TestDetectStartupModeEnvironmentAloneRemainsDemo(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")
	setGopherEnvValue("GOPHER_ENVIRONMENT", "development")

	if got := detectStartupMode(); got != "demo" {
		t.Fatalf("detectStartupMode() = %q, want demo without a repository", got)
	}
}

func TestConfigTemplateEnvironmentHasNoProductionFallback(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveDeployEnvironment(t)
	unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")

	if got := currentConfigTemplateEnvironment(); got != "" {
		t.Fatalf("currentConfigTemplateEnvironment() = %q, want empty", got)
	}
}

func TestLoadConfigVariablesAllowsEnvironmentlessDemo(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	preserveDeployEnvironment(t)
	unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
	unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")

	values, err := loadConfigVariables()
	if err != nil {
		t.Fatalf("loadConfigVariables() error = %v, want environmentless demo to succeed", err)
	}
	if len(values.Secrets) != 0 || len(values.Variables) != 0 {
		t.Fatalf("environmentless demo variables = %+v, want empty", values)
	}
}

func TestLoadConfigVariablesRequiresEnvironmentForCLIConfig(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	preserveDeployEnvironment(t)
	unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
	cliOp = true

	_, err := loadConfigVariables()
	if err == nil || !strings.Contains(err.Error(), "required and has no default") {
		t.Fatalf("loadConfigVariables() error = %v, want required environment error", err)
	}
}

func TestCliGenKeyRequiresEnvironmentOrFlag(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveDeployEnvironment(t)
	unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
	setGopherEnvValue("GOPHER_ENCRYPTION_KEY", "12345678901234567890123456789012")

	err := cliGenKey("", false, false)
	if err == nil || !strings.Contains(err.Error(), "required and has no default") {
		t.Fatalf("cliGenKey() error = %v, want required environment error", err)
	}
}

func preserveDeployEnvironment(t *testing.T) {
	t.Helper()
	old := deployEnvironment
	deployEnvironment = ""
	t.Cleanup(func() {
		deployEnvironment = old
	})
}
