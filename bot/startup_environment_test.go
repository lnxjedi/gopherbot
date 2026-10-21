package bot

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestInitializeStartupEnvironmentMatrix(t *testing.T) {
	tests := []struct {
		name            string
		unset           bool
		environment     string
		wantEnvironment string
		wantError       string
	}{
		{name: "unset defaults to production", unset: true, wantEnvironment: "production"},
		{name: "empty defaults to production", wantEnvironment: "production"},
		{name: "whitespace defaults to production", environment: "   ", wantEnvironment: "production"},
		{name: "explicit value is normalized", environment: " development ", wantEnvironment: "development"},
		{name: "configured environment must be valid", environment: "../production", wantError: "single path segment"},
		{name: "configured environment is accepted", environment: "production", wantEnvironment: "production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preserveGopherEnvMaps(t)
			unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
			if !tt.unset {
				setGopherEnvValue("GOPHER_ENVIRONMENT", tt.environment)
			}
			err := initializeStartupEnvironment()
			if tt.wantError == "" && err != nil {
				t.Fatalf("initializeStartupEnvironment() error = %v, want nil", err)
			}
			if tt.wantError != "" && (err == nil || !strings.Contains(err.Error(), tt.wantError)) {
				t.Fatalf("initializeStartupEnvironment() error = %v, want substring %q", err, tt.wantError)
			}
			if err == nil && getEnv("GOPHER_ENVIRONMENT") != tt.wantEnvironment {
				t.Fatalf("GOPHER_ENVIRONMENT = %q, want %q", getEnv("GOPHER_ENVIRONMENT"), tt.wantEnvironment)
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

func TestConfigTemplateEnvironmentDefaultsToProduction(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveDeployEnvironment(t)
	unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")

	if got := currentConfigTemplateEnvironment(); got != "production" {
		t.Fatalf("currentConfigTemplateEnvironment() = %q, want production", got)
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

func TestLoadConfigVariablesDefaultsToProductionForCLIConfig(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	preserveDeployEnvironment(t)
	unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
	cliOp = true
	varDir := filepath.Join(configPath, "conf", "variables")
	if err := os.MkdirAll(varDir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, environment := range []string{"common", "production", "development"} {
		if err := os.WriteFile(filepath.Join(varDir, environment+".yaml"), []byte("Variables:\n  SOURCE: "+environment+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	values, err := loadConfigVariables()
	if err != nil {
		t.Fatal(err)
	}
	if got := values.Variables["SOURCE"]; got != "production" {
		t.Fatalf("SOURCE = %q, want production", got)
	}
}

func TestCliGenKeyDefaultsToProductionAndAllowsExplicitEnvironment(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	preserveDeployEnvironment(t)
	unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
	setGopherEnvValue("GOPHER_ENCRYPTION_KEY", "12345678901234567890123456789012")

	for _, environment := range []string{"", "development"} {
		if err := cliGenKey(environment, true, false); err != nil {
			t.Fatalf("cliGenKey(%q) error = %v", environment, err)
		}
		filename := encryptedKeyFile
		if environment != "" {
			filename += "." + environment
		}
		if _, err := os.Stat(filepath.Join(configPath, filename)); err != nil {
			t.Fatalf("generated key %s: %v", filename, err)
		}
	}
}

func TestStartupEnvironmentDefaultNoticeIsInfoAndEmittedOnce(t *testing.T) {
	for _, level := range []robot.LogLevel{robot.Info, robot.Warn} {
		t.Run(logLevelToStr(level), func(t *testing.T) {
			preserveGopherEnvMaps(t)
			unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
			state := snapshotLogState()
			var output bytes.Buffer
			botLogger.Lock()
			oldLogger, oldLevel := botLogger.logger, botLogger.level
			botLogger.logger, botLogger.level = log.New(&output, "", 0), level
			botLogger.Unlock()
			t.Cleanup(func() {
				botLogger.Lock()
				botLogger.logger, botLogger.level = oldLogger, oldLevel
				botLogger.Unlock()
				restoreLogState(state)
			})
			for i := 0; i < 2; i++ {
				if err := initializeStartupEnvironment(); err != nil {
					t.Fatal(err)
				}
			}
			want := ""
			if level == robot.Info {
				want = "Info: GOPHER_ENVIRONMENT isn't set, defaulting to 'production'\n"
			}
			if output.String() != want {
				t.Fatalf("startup log = %q, want %q", output.String(), want)
			}
			w := &worker{Incoming: &robot.ConnectorMessage{}, cfg: &configuration{}, pipeContext: &pipeContext{}}
			env, _ := w.getEnvironment(&Task{})
			if got := env["GOPHER_ENVIRONMENT"]; got != "production" {
				t.Fatalf("extension environment = %q, want production", got)
			}
		})
	}
}

func TestStartupEnvironmentPreservesPrivateFileSelection(t *testing.T) {
	preserveGopherEnvMaps(t)
	unsetProcessEnvForTest(t, "GOPHER_ENVIRONMENT")
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("GOPHER_ENVIRONMENT=development\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadPrivateEnvironment(envFile); err != nil {
		t.Fatal(err)
	}
	if err := initializeStartupEnvironment(); err != nil {
		t.Fatal(err)
	}
	if got := getEnv("GOPHER_ENVIRONMENT"); got != "development" {
		t.Fatalf("GOPHER_ENVIRONMENT = %q, want development from private file", got)
	}
	if got := os.Getenv("GOPHER_ENVIRONMENT"); got != "" {
		t.Fatalf("GOPHER_ENVIRONMENT leaked into process environment: %q", got)
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
