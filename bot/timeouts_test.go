package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func testConfigDurationValue(d time.Duration) ConfigDuration {
	return ConfigDuration{duration: d, set: true}
}

func TestResolveTimeOutThresholdsExplicitZeroDisablesDefaults(t *testing.T) {
	defaults := runtimeTimeOutThresholds{
		Warn: 7 * time.Minute,
		Kill: 14 * time.Minute,
	}
	overrides := TimeOutThresholds{
		Warn: testConfigDurationValue(0),
		Kill: testConfigDurationValue(0),
	}

	resolved := resolveTimeOutThresholds(defaults, overrides)
	if resolved.Warn != 0 || resolved.Kill != 0 {
		t.Fatalf("resolveTimeOutThresholds() = %+v, want both thresholds disabled", resolved)
	}
}

func TestResolveTimeOutThresholdsUsesExplicitOverrides(t *testing.T) {
	defaults := runtimeTimeOutThresholds{
		Warn: 7 * time.Minute,
		Kill: 14 * time.Minute,
	}
	overrides := TimeOutThresholds{
		Warn: testConfigDurationValue(90 * time.Second),
	}

	resolved := resolveTimeOutThresholds(defaults, overrides)
	if resolved.Warn != 90*time.Second {
		t.Fatalf("resolved Warn = %v, want %v", resolved.Warn, 90*time.Second)
	}
	if resolved.Kill != defaults.Kill {
		t.Fatalf("resolved Kill = %v, want default %v", resolved.Kill, defaults.Kill)
	}
}

func TestValidateRuntimeTimeOutThresholdsRejectsEffectiveKillNotGreaterThanWarn(t *testing.T) {
	err := validateRuntimeTimeOutThresholds("task 'slow' effective TimeOuts", runtimeTimeOutThresholds{
		Warn: 3 * time.Minute,
		Kill: 2 * time.Minute,
	})
	if err == nil {
		t.Fatalf("expected invalid effective timeout validation failure")
	}
	if !strings.Contains(err.Error(), "Kill must be greater than Warn") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestInstalledPluginTimeOutsByStartupMode(t *testing.T) {
	installedRobotConfig, err := os.ReadFile(filepath.Join("..", "conf", robotConfigFileName))
	if err != nil {
		t.Fatalf("read installed robot config: %v", err)
	}

	tests := []struct {
		name       string
		configured bool
		warn       time.Duration
		kill       time.Duration
	}{
		{name: "demo", warn: 35 * time.Minute, kill: 42 * time.Minute},
		{name: "bootstrap", configured: true, warn: 7 * time.Minute, kill: 14 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preserveGopherEnvMaps(t)
			preserveStartupModeGlobals(t)
			unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")
			if tt.configured {
				setGopherEnvValue("GOPHER_CUSTOM_REPOSITORY", "git@example.com:robots/example-robot.git")
			}

			expanded, err := expand("conf", false, installedRobotConfig)
			if err != nil {
				t.Fatalf("expand installed robot config: %v", err)
			}
			var cfg struct {
				TimeOuts TimeOutsConfig `yaml:"TimeOuts"`
			}
			if err := yaml.Unmarshal(expanded, &cfg); err != nil {
				t.Fatalf("parse expanded installed robot config: %v", err)
			}
			if got := cfg.TimeOuts.Plugin.Warn.Duration(); got != tt.warn {
				t.Errorf("Plugin.Warn = %v, want %v", got, tt.warn)
			}
			if got := cfg.TimeOuts.Plugin.Kill.Duration(); got != tt.kill {
				t.Errorf("Plugin.Kill = %v, want %v", got, tt.kill)
			}
			if got := cfg.TimeOuts.Job.Warn.Duration(); got != time.Hour {
				t.Errorf("Job.Warn = %v, want %v", got, time.Hour)
			}
			if got := cfg.TimeOuts.Job.Kill.Duration(); got != 2*time.Hour {
				t.Errorf("Job.Kill = %v, want %v", got, 2*time.Hour)
			}
		})
	}
}

func TestRobotSkeletonKeepsExplicitPluginTimeOuts(t *testing.T) {
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	oldVariables := activeConfigVariables.values
	t.Cleanup(func() { setActiveConfigVariables(oldVariables) })
	unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")

	skeletonPath, err := filepath.Abs(filepath.Join("..", "robot.skel"))
	if err != nil {
		t.Fatalf("resolve robot skeleton path: %v", err)
	}
	configPath = skeletonPath
	setGopherEnvValue("GOPHER_CUSTOM_REPOSITORY", "git@example.com:robots/example-robot.git")
	setGopherEnvValue("GOPHER_ENVIRONMENT", "production")
	variables, err := loadConfigVariables()
	if err != nil {
		t.Fatalf("load Robot skeleton variables: %v", err)
	}
	setActiveConfigVariables(variables)

	robotConfig, err := os.ReadFile(filepath.Join(configPath, "conf", robotConfigFileName))
	if err != nil {
		t.Fatalf("read Robot skeleton config: %v", err)
	}
	expanded, err := expand("conf", true, robotConfig)
	if err != nil {
		t.Fatalf("expand Robot skeleton config: %v", err)
	}
	var cfg struct {
		TimeOuts TimeOutsConfig `yaml:"TimeOuts"`
	}
	if err := yaml.Unmarshal(expanded, &cfg); err != nil {
		t.Fatalf("parse expanded Robot skeleton config: %v", err)
	}
	if got := cfg.TimeOuts.Plugin.Warn.Duration(); got != 7*time.Minute {
		t.Errorf("Plugin.Warn = %v, want %v", got, 7*time.Minute)
	}
	if got := cfg.TimeOuts.Plugin.Kill.Duration(); got != 14*time.Minute {
		t.Errorf("Plugin.Kill = %v, want %v", got, 14*time.Minute)
	}
}

func preserveStartupModeGlobals(t *testing.T) {
	t.Helper()
	oldCliOp := cliOp
	oldConfigPath := configPath
	cliOp = false
	configPath = t.TempDir()
	t.Cleanup(func() {
		cliOp = oldCliOp
		configPath = oldConfigPath
	})
}

func unsetProcessEnvForTest(t *testing.T, key string) {
	t.Helper()
	oldValue, wasSet := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if wasSet {
			if err := os.Setenv(key, oldValue); err != nil {
				t.Errorf("restore %s: %v", key, err)
			}
			return
		}
		if err := os.Unsetenv(key); err != nil {
			t.Errorf("clear %s after test: %v", key, err)
		}
	})
}
