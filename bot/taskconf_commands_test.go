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
