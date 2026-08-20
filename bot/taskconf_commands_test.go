package bot

import (
	"strings"
	"testing"
)

func TestValidateYAMLPluginAcceptsCommandsKey(t *testing.T) {
	yml := []byte(`
---
Commands:
- Command: ping
  Regex: '(?i:ping)'
`)
	if err := validate_yaml("conf/plugins/example.yaml", yml); err != nil {
		t.Fatalf("validate_yaml() rejected Commands key: %v", err)
	}
}

func TestValidateYAMLPluginAcceptsSimpleMatcherInCommands(t *testing.T) {
	yml := []byte(`
---
Commands:
- Command: ping
  SimpleMatcher: ping
`)
	if err := validate_yaml("conf/plugins/example.yaml", yml); err != nil {
		t.Fatalf("validate_yaml() rejected SimpleMatcher: %v", err)
	}
}

func TestValidateYAMLPluginAcceptsMultilineCommandDetails(t *testing.T) {
	yml := []byte(`
---
Commands:
- Command: deploy
  SimpleMatcher: deploy
  Details: |
    Deploys the current build.

    **Notes**

    This may take several minutes.
`)
	if err := validate_yaml("conf/plugins/example.yaml", yml); err != nil {
		t.Fatalf("validate_yaml() rejected Details: %v", err)
	}
}

func TestValidateYAMLPluginRejectsLegacyCommandMatchersKey(t *testing.T) {
	yml := []byte(`
---
CommandMatchers:
- Command: ping
  Regex: '(?i:ping)'
`)
	err := validate_yaml("conf/plugins/example.yaml", yml)
	if err == nil {
		t.Fatalf("validate_yaml() accepted legacy CommandMatchers key")
	}
	if !strings.Contains(err.Error(), "CommandMatchers") {
		t.Fatalf("validate_yaml() error %q did not reference CommandMatchers", err)
	}
}

func TestValidateYAMLPluginRejectsLegacyHelpKey(t *testing.T) {
	yml := []byte(`
---
Help:
- Keywords: [ "ping" ]
  Helptext: [ "(alias) ping - test" ]
Commands:
- Command: ping
  Regex: '(?i:ping)'
`)
	err := validate_yaml("conf/plugins/example.yaml", yml)
	if err == nil {
		t.Fatalf("validate_yaml() accepted legacy Help key")
	}
	if !strings.Contains(err.Error(), "Help") {
		t.Fatalf("validate_yaml() error %q did not reference Help", err)
	}
}

func TestValidateYAMLPluginRejectsHelptextInCommands(t *testing.T) {
	yml := []byte(`
---
Commands:
- Command: ping
  Regex: '(?i:ping)'
  Helptext: [ "(alias) ping - test" ]
`)
	err := validate_yaml("conf/plugins/example.yaml", yml)
	if err == nil {
		t.Fatalf("validate_yaml() accepted Helptext in Commands")
	}
	if !strings.Contains(err.Error(), "Helptext") {
		t.Fatalf("validate_yaml() error %q did not reference Helptext", err)
	}
}

func TestValidateYAMLPluginRejectsEngineReservedCommand(t *testing.T) {
	yml := []byte(`
---
Commands:
- Command: _subscribed
  Regex: '(?i:subscribed)'
`)
	err := validate_yaml("conf/plugins/example.yaml", yml)
	if err == nil {
		t.Fatalf("validate_yaml() accepted engine-reserved command")
	}
	if !strings.Contains(err.Error(), "reserved for engine use") || !strings.Contains(err.Error(), "_") {
		t.Fatalf("validate_yaml() error %q did not explain reserved command namespace", err)
	}
}

func TestValidateYAMLPluginRejectsEngineReservedMessageMatcherCommand(t *testing.T) {
	yml := []byte(`
---
MessageMatchers:
- Command: _ambient
  Regex: '(?i:.*)'
`)
	err := validate_yaml("conf/plugins/example.yaml", yml)
	if err == nil {
		t.Fatalf("validate_yaml() accepted engine-reserved message matcher command")
	}
	if !strings.Contains(err.Error(), "reserved for engine use") || !strings.Contains(err.Error(), "_") {
		t.Fatalf("validate_yaml() error %q did not explain reserved command namespace", err)
	}
}
