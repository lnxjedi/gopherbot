package bot

import (
	"encoding/json"
	"testing"
)

func TestNormalizePluginCommandMatcherKeysFromCommands(t *testing.T) {
	cfg := map[string]json.RawMessage{
		"Commands": json.RawMessage(`[{"Command":"newkey","Regex":"(?i:new)"}]`),
	}

	normalizePluginCommandMatcherKeys("demo", cfg)

	if _, ok := cfg["Commands"]; ok {
		t.Fatalf("normalizePluginCommandMatcherKeys() did not remove Commands key")
	}
	got, ok := cfg["CommandMatchers"]
	if !ok {
		t.Fatalf("normalizePluginCommandMatcherKeys() did not set CommandMatchers")
	}
	if string(got) != `[{"Command":"newkey","Regex":"(?i:new)"}]` {
		t.Fatalf("normalizePluginCommandMatcherKeys() CommandMatchers = %s", got)
	}
}

func TestNormalizePluginCommandMatcherKeysPrefersCommands(t *testing.T) {
	cfg := map[string]json.RawMessage{
		"Commands":        json.RawMessage(`[{"Command":"newkey","Regex":"(?i:new)"}]`),
		"CommandMatchers": json.RawMessage(`[{"Command":"legacy","Regex":"(?i:legacy)"}]`),
	}

	normalizePluginCommandMatcherKeys("demo", cfg)

	got, ok := cfg["CommandMatchers"]
	if !ok {
		t.Fatalf("normalizePluginCommandMatcherKeys() did not set CommandMatchers")
	}
	if string(got) != `[{"Command":"newkey","Regex":"(?i:new)"}]` {
		t.Fatalf("normalizePluginCommandMatcherKeys() did not prefer Commands, got %s", got)
	}
	if _, ok := cfg["Commands"]; ok {
		t.Fatalf("normalizePluginCommandMatcherKeys() did not remove Commands key")
	}
}

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

func TestBuildLegacyHelpFromCommandMetadata(t *testing.T) {
	matchers := []InputMatcher{
		{
			Command:  "ping",
			Keywords: []string{"ping", "latency"},
			Helptext: []string{"(alias) ping - see if the bot is alive"},
		},
		{
			Command:  "help",
			Helptext: []string{"(alias) help <keyword> - find help"},
		},
		{
			Command: "ignore",
		},
		{
			Command: "info",
			Usage:   "(alias) info",
			Summary: "show robot info",
		},
	}

	help := buildLegacyHelpFromCommandMetadata(matchers)
	if len(help) != 2 {
		t.Fatalf("buildLegacyHelpFromCommandMetadata() len = %d, want 2", len(help))
	}
	if len(help[0].Keywords) != 2 || help[0].Keywords[0] != "ping" || help[0].Keywords[1] != "latency" {
		t.Fatalf("buildLegacyHelpFromCommandMetadata() first keywords = %#v", help[0].Keywords)
	}
	if len(help[1].Keywords) != 1 || help[1].Keywords[0] != "help" {
		t.Fatalf("buildLegacyHelpFromCommandMetadata() second keywords = %#v, want fallback [help]", help[1].Keywords)
	}
}
