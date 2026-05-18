package slack

import (
	"io"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

type testHandler struct {
	protocolConfig *config
}

func (t *testHandler) IncomingMessage(_ *robot.ConnectorMessage) {}
func (t *testHandler) GetProtocolConfig(v interface{}) error {
	if t.protocolConfig != nil {
		*(v.(*config)) = *t.protocolConfig
	}
	return nil
}
func (t *testHandler) GetBrainConfig(_ interface{}) error         { return nil }
func (t *testHandler) GetEventStrings() *[]string                 { return nil }
func (t *testHandler) GetHistoryConfig(_ interface{}) error       { return nil }
func (t *testHandler) GetBotInfo() robot.BotInfo                  { return robot.BotInfo{} }
func (t *testHandler) SetBotID(_ string)                          {}
func (t *testHandler) SetTerminalWriter(_ io.Writer)              {}
func (t *testHandler) SetBotMention(_ string)                     {}
func (t *testHandler) GetLogLevel() robot.LogLevel                { return robot.Info }
func (t *testHandler) GetInstallPath() string                     { return "" }
func (t *testHandler) GetConfigPath() string                      { return "" }
func (t *testHandler) ReadEncryptedFile(_ string) ([]byte, error) { return nil, nil }
func (t *testHandler) Log(_ robot.LogLevel, _ string, _ ...interface{}) {
}
func (t *testHandler) GetDirectory(_ string) error { return nil }

func TestNormalizeConfiguredUserMap(t *testing.T) {
	h := &testHandler{}
	got := normalizeConfiguredUserMap(map[string]string{
		"alice": " U0001 ",
		"Bob":   "U0002",
		"carol": "   ",
		"":      "U0004",
		"david": "U0005",
	}, h)

	if len(got) != 2 {
		t.Fatalf("expected 2 valid mappings, got %d (%#v)", len(got), got)
	}
	if got["alice"] != "U0001" {
		t.Fatalf("expected alice -> U0001, got %q", got["alice"])
	}
	if got["david"] != "U0005" {
		t.Fatalf("expected david -> U0005, got %q", got["david"])
	}
	if _, ok := got["Bob"]; ok {
		t.Fatalf("expected uppercase username entry to be rejected")
	}
	if _, ok := got["carol"]; ok {
		t.Fatalf("expected empty-id entry to be rejected")
	}
}

func TestNormalizeConfiguredUserMapEmpty(t *testing.T) {
	h := &testHandler{}
	if got := normalizeConfiguredUserMap(nil, h); got != nil {
		t.Fatalf("expected nil map for nil input")
	}
	if got := normalizeConfiguredUserMap(map[string]string{}, h); got != nil {
		t.Fatalf("expected nil map for empty input")
	}
}

func TestReloadAtomicallySwapsConfiguredUserMap(t *testing.T) {
	h := &testHandler{
		protocolConfig: &config{UserMap: map[string]string{
			"bob":   "U22",
			"david": "U4",
		}},
	}
	sc := &slackConnector{
		Handler:    h,
		botUserMap: map[string]string{"alice": "U1"},
		userMap: map[string]string{
			"alice": "U1",
			"bob":   "U2",
			"carol": "U3",
		},
		userIDMap: map[string]string{
			"U1": "alice",
			"U2": "bob",
			"U3": "carol",
		},
	}

	if err := sc.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if _, ok := sc.botUserMap["alice"]; ok {
		t.Fatalf("expected removed configured user alice to be absent from botUserMap")
	}
	if sc.botUserMap["bob"] != "U22" || sc.botUserMap["david"] != "U4" {
		t.Fatalf("unexpected botUserMap after reload: %#v", sc.botUserMap)
	}
	if _, ok := sc.userMap["alice"]; ok {
		t.Fatalf("expected old configured user alice to be removed from userMap")
	}
	if sc.userMap["bob"] != "U22" || sc.userIDMap["U22"] != "bob" {
		t.Fatalf("expected bob mapping to be replaced atomically, userMap=%#v userIDMap=%#v", sc.userMap, sc.userIDMap)
	}
	if sc.userMap["carol"] != "U3" || sc.userIDMap["U3"] != "carol" {
		t.Fatalf("expected non-configured user map entries to be preserved")
	}
}

func TestResolveSlashCommandConfig(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name    string
		cfg     config
		wantOK  bool
		wantCmd string
	}{
		{
			name:   "missing explicit accept",
			cfg:    config{},
			wantOK: false,
		},
		{
			name: "disabled slash commands",
			cfg:  config{AcceptSlashCommands: &falseVal},
		},
		{
			name:   "enabled requires command",
			cfg:    config{AcceptSlashCommands: &trueVal},
			wantOK: false,
		},
		{
			name:    "enabled normalizes slash",
			cfg:     config{AcceptSlashCommands: &trueVal, SlashCommand: " /clu "},
			wantOK:  true,
			wantCmd: "clu",
		},
	}

	for _, tt := range tests {
		enabled, command, err := resolveSlashCommandConfig(tt.cfg)
		if tt.wantOK {
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tt.name, err)
			}
			if !enabled {
				t.Fatalf("%s: expected enabled", tt.name)
			}
			if command != tt.wantCmd {
				t.Fatalf("%s: command = %q, want %q", tt.name, command, tt.wantCmd)
			}
			continue
		}
		if tt.cfg.AcceptSlashCommands != nil && !*tt.cfg.AcceptSlashCommands {
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tt.name, err)
			}
			if enabled {
				t.Fatalf("%s: expected disabled", tt.name)
			}
			if command != "" {
				t.Fatalf("%s: command = %q, want empty", tt.name, command)
			}
			continue
		}
		if err == nil {
			t.Fatalf("%s: expected error", tt.name)
		}
	}
}
