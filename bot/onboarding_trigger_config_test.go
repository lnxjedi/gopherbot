package bot

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type onboardingTriggerConfig struct {
	Triggers []struct {
		User    string `yaml:"User"`
		Channel string `yaml:"Channel"`
		Regex   string `yaml:"Regex"`
	} `yaml:"Triggers"`
}

func TestInstalledRobotUsesFixedDefaultIdentity(t *testing.T) {
	resetConfigVariableTestState(t)
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")
	setGopherEnvValue("GOPHER_BOTNAME", "example-robot")
	setGopherEnvValue("GOPHER_BOTFULLNAME", "Example Robot")

	raw, err := os.ReadFile(filepath.Join("..", "conf", robotConfigFileName))
	if err != nil {
		t.Fatalf("read installed Robot config: %v", err)
	}
	expanded, err := expand("conf", false, raw)
	if err != nil {
		t.Fatalf("expand installed Robot config: %v", err)
	}
	var cfg struct {
		BotInfo struct {
			UserName string `yaml:"UserName"`
			FullName string `yaml:"FullName"`
		} `yaml:"BotInfo"`
	}
	if err := yaml.Unmarshal(expanded, &cfg); err != nil {
		t.Fatalf("parse installed Robot config: %v", err)
	}
	if got := cfg.BotInfo.UserName; got != "floyd" {
		t.Errorf("BotInfo.UserName = %q, want floyd", got)
	}
	if got := cfg.BotInfo.FullName; got != "Floyd Gopherbot" {
		t.Errorf("BotInfo.FullName = %q, want Floyd Gopherbot", got)
	}
}

func TestInstalledRobotUsesFixedDefaultAlias(t *testing.T) {
	resetConfigVariableTestState(t)
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")
	setGopherEnvValue("GOPHER_ALIAS", "!")

	raw, err := os.ReadFile(filepath.Join("..", "conf", robotConfigFileName))
	if err != nil {
		t.Fatalf("read installed Robot config: %v", err)
	}
	expanded, err := expand("conf", false, raw)
	if err != nil {
		t.Fatalf("expand installed Robot config: %v", err)
	}
	var cfg struct {
		Alias string `yaml:"Alias"`
	}
	if err := yaml.Unmarshal(expanded, &cfg); err != nil {
		t.Fatalf("parse installed Robot config: %v", err)
	}
	if got := cfg.Alias; got != ";" {
		t.Errorf("Alias = %q, want fixed default alias ;", got)
	}
}

func TestInstalledOnboardingTriggersUseDefaultRobotNameInDemo(t *testing.T) {
	resetConfigVariableTestState(t)
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")
	setGopherEnvValue("GOPHER_BOTNAME", "example-robot")

	for _, name := range []string{"welcome-join", "resume-setup"} {
		t.Run(name, func(t *testing.T) {
			cfg := expandInstalledOnboardingTrigger(t, name)
			if got := cfg.Triggers[0].User; got != "floyd" {
				t.Errorf("trigger User = %q, want fixed demo identity floyd", got)
			}
		})
	}
}

func TestInstalledOnboardingTriggersUseConfiguredRobotVariable(t *testing.T) {
	resetConfigVariableTestState(t)
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	setGopherEnvValue("GOPHER_CUSTOM_REPOSITORY", "git@example.com:robots/acme-bot.git")
	setGopherEnvValue("GOPHER_ENVIRONMENT", "production")

	confDir := filepath.Join(configPath, "conf")
	if err := os.MkdirAll(confDir, 0700); err != nil {
		t.Fatalf("create configured conf directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(confDir, robotConfigFileName), []byte("Alias: ;\n"), 0600); err != nil {
		t.Fatalf("write configured robot file: %v", err)
	}
	setActiveConfigVariables(&configVariableSet{
		Secrets: map[string]string{},
		Variables: map[string]string{
			"ROBOT_NAME": "acme-bot",
		},
	})

	for _, name := range []string{"welcome-join", "resume-setup"} {
		t.Run(name, func(t *testing.T) {
			cfg := expandInstalledOnboardingTrigger(t, name)
			if got := cfg.Triggers[0].User; got != "acme-bot" {
				t.Errorf("trigger User = %q, want configured ROBOT_NAME", got)
			}
		})
	}
}

func expandInstalledOnboardingTrigger(t *testing.T, name string) onboardingTriggerConfig {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "conf", "jobs", name+".yaml"))
	if err != nil {
		t.Fatalf("read installed %s config: %v", name, err)
	}
	expanded, err := expand(filepath.Join("conf", "jobs"), false, raw)
	if err != nil {
		t.Fatalf("expand installed %s config: %v", name, err)
	}
	var cfg onboardingTriggerConfig
	if err := yaml.Unmarshal(expanded, &cfg); err != nil {
		t.Fatalf("parse installed %s config: %v", name, err)
	}
	if len(cfg.Triggers) != 1 {
		t.Fatalf("installed %s trigger count = %d, want 1", name, len(cfg.Triggers))
	}
	if cfg.Triggers[0].Channel == "" || cfg.Triggers[0].Regex == "" {
		t.Fatalf("installed %s trigger lost channel or regex: %+v", name, cfg.Triggers[0])
	}
	return cfg
}
