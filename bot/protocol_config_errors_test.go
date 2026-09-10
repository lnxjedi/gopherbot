package bot

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func preserveProtocolConfigTestState(t *testing.T) {
	t.Helper()
	protocolConfigs.RLock()
	configs, loadErrors := protocolConfigs.m, protocolConfigs.errors
	protocolConfigs.RUnlock()
	t.Cleanup(func() { setProtocolConfigs(configs, loadErrors) })
}

func writeProtocolConfigTestFile(t *testing.T, base, content string) {
	t.Helper()
	dir := filepath.Join(base, "conf", "protocols")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "slack.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestProtocolConfigIsolationAndRecovery(t *testing.T) {
	preserveProtocolConfigTestState(t)
	h := newRuntimeHarness(t)
	h.setConfig("ssh", "slack")
	h.registerFake("ssh")
	connectorRegistrationOverrides["slack"] = robot.ConnectorRegistration{
		Initialize: func(handler robot.Handler, _ *log.Logger) robot.InitializedConnector {
			var cfg struct{ AcceptSlashCommands *bool }
			if err := handler.GetProtocolConfig(&cfg); err != nil {
				return robot.InitializedConnector{Error: err}
			}
			if cfg.AcceptSlashCommands == nil || !*cfg.AcceptSlashCommands {
				return robot.InitializedConnector{Error: errors.New("missing slash setting")}
			}
			return robot.InitializedConnector{Connector: &fakeRuntimeConnector{}}
		},
	}
	configs := map[string]json.RawMessage{"ssh": json.RawMessage(`{"ListenPort":2222}`)}
	setProtocolConfigs(configs, nil)
	var decoded map[string]interface{}
	if err := (connectorHandler{protocol: "slack"}).GetProtocolConfig(&decoded); err == nil || decoded != nil {
		t.Fatalf("missing Slack config: err=%v, decoded=%v; must not receive SSH config", err, decoded)
	}
	if err := handle.GetProtocolConfig(&decoded); err != nil || decoded["ListenPort"] != float64(2222) {
		t.Fatalf("primary config: err=%v, decoded=%v", err, decoded)
	}
	loadErr := errors.New("loading secondary protocol config from conf/protocols/slack.yaml: duplicate mapping key")
	setProtocolConfigs(configs, map[string]error{" SLACK ": loadErr})
	if err := (connectorHandler{protocol: "slack"}).GetProtocolConfig(&decoded); !errors.Is(err, loadErr) {
		t.Fatalf("config error = %v, want original load error", err)
	}
	if err := initializeConnectorRuntime(log.New(io.Discard, "", 0)); err != nil {
		t.Fatalf("secondary config error aborted primary initialization: %v", err)
	}
	if got := statusMap()["slack"]; got.state != "failed" || got.err != loadErr.Error() {
		t.Fatalf("secondary status = %+v, want original config error", got)
	}
	configs["slack"] = json.RawMessage(`{"AcceptSlashCommands":true}`)
	setProtocolConfigs(configs, nil)
	if err := ensureConnectorInitialized("slack", false, log.New(io.Discard, "", 0)); err != nil {
		t.Fatalf("retry with corrected configuration failed: %v", err)
	}
	if got := statusMap()["slack"]; got.err != "" || got.state == "failed" {
		t.Fatalf("recovered secondary status = %+v", got)
	}
}

func TestLoadProtocolFileDataPreservesErrors(t *testing.T) {
	for _, tc := range []struct {
		name, content, want string
	}{
		{"duplicate", "ProtocolConfig:\n  AcceptSlashCommands: true\n  UserMap:\n    alice: U123\n    alice: U123\n", `mapping key "alice" already defined`},
		{"template", "ProtocolConfig:\n  AcceptSlashCommands: {{ variable \"UNDEFINED\" }}\n", `template variable "UNDEFINED" is not defined`},
		{"missing file", "", "no such file or directory"},
		{"missing section", "ChannelRoster: []\n", "has no ProtocolConfig"},
		{"null section", "ProtocolConfig: null\n", "has no ProtocolConfig"},
		{"misplaced mapping", "UserMap:\n  alice: U123\nProtocolConfig: {}\n", "field UserMap not found"},
		{"serialization", "ProtocolConfig:\n  MaxMessageSplit: .nan\n", "serializing configuration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigVariableTestState(t)
			installPath, configPath = t.TempDir(), t.TempDir()
			if tc.content != "" {
				writeProtocolConfigTestFile(t, configPath, tc.content)
			}
			for _, role := range []string{"primary", "secondary"} {
				_, err := loadProtocolFileData(&ConfigLoader{}, "slack", role)
				if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "protocols/slack.yaml") {
					t.Fatalf("%s load error = %v, want path and %q", role, err, tc.want)
				}
			}
		})
	}
}

func TestGetConfigFileRejectsBrokenLayers(t *testing.T) {
	for _, layer := range []string{"installed", "custom"} {
		for _, required := range []bool{false, true} {
			name := layer + "/optional"
			if required {
				name = layer + "/required"
			}
			t.Run(name, func(t *testing.T) {
				resetConfigVariableTestState(t)
				installPath, configPath = t.TempDir(), t.TempDir()
				valid := "ProtocolConfig:\n  AcceptSlashCommands: true\n"
				writeProtocolConfigTestFile(t, installPath, valid)
				writeProtocolConfigTestFile(t, configPath, valid)
				brokenPath := installPath
				if layer == "custom" {
					brokenPath = configPath
				}
				writeProtocolConfigTestFile(t, brokenPath, "ProtocolConfig:\n  BotToken: {{ variable \"UNDEFINED\" }}\n")
				cfg := make(map[string]json.RawMessage)
				err := getConfigFile("protocols/slack.yaml", required, cfg)
				if err == nil || !strings.Contains(err.Error(), `template variable "UNDEFINED" is not defined`) {
					t.Fatalf("broken %s layer fell back to valid layer: %v", layer, err)
				}
			})
		}
	}
}

func TestGetConfigFileMissingAndUnreadableLayer(t *testing.T) {
	resetConfigVariableTestState(t)
	installPath, configPath = t.TempDir(), t.TempDir()
	writeProtocolConfigTestFile(t, installPath, "ProtocolConfig:\n  AcceptSlashCommands: false\n  MaxMessageSplit: 2\n")
	read := func() (map[string]json.RawMessage, error) {
		cfg := make(map[string]json.RawMessage)
		err := getConfigFile("protocols/slack.yaml", true, cfg)
		return cfg, err
	}
	if _, err := read(); err != nil {
		t.Fatalf("missing custom layer should use installed defaults: %v", err)
	}
	writeProtocolConfigTestFile(t, configPath, "ProtocolConfig:\n  AcceptSlashCommands: true\n")
	cfg, err := read()
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		AcceptSlashCommands bool
		MaxMessageSplit     int
	}
	if err := json.Unmarshal(cfg["ProtocolConfig"], &decoded); err != nil || !decoded.AcceptSlashCommands || decoded.MaxMessageSplit != 2 {
		t.Fatalf("custom override and installed defaults: %+v, err=%v", decoded, err)
	}
	path := filepath.Join(configPath, "conf", "protocols", "slack.yaml")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := read(); err == nil || !strings.Contains(err.Error(), "loading custom configuration") {
		t.Fatalf("unreadable custom layer should fail, got: %v", err)
	}
}
