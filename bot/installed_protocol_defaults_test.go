package bot

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInstalledProtocolDefaultsContainOnlyProtocolConfig(t *testing.T) {
	resetConfigVariableTestState(t)
	preserveGopherEnvMaps(t)
	preserveStartupModeGlobals(t)
	unsetProcessEnvForTest(t, "GOPHER_CUSTOM_REPOSITORY")

	for _, protocol := range []string{"nullconn", "ssh", "terminal"} {
		t.Run(protocol, func(t *testing.T) {
			path := filepath.Join("..", "conf", "protocols", protocol+".yaml")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read installed %s defaults: %v", protocol, err)
			}
			for _, retired := range []string{
				"GOPHER_BOTNAME",
				"GOPHER_BOTFULLNAME",
				"GOPHER_ALIAS",
				"GOPHER_JOBCHANNEL",
			} {
				if strings.Contains(string(raw), retired) {
					t.Errorf("installed %s defaults retain ignored lookup %s", protocol, retired)
				}
			}

			expanded, err := expand(filepath.Join("conf", "protocols"), false, raw)
			if err != nil {
				t.Fatalf("expand installed %s defaults: %v", protocol, err)
			}
			var cfg map[string]yaml.Node
			if err := yaml.Unmarshal(expanded, &cfg); err != nil {
				t.Fatalf("parse installed %s defaults: %v", protocol, err)
			}
			if len(cfg) != 1 {
				t.Fatalf("installed %s top-level keys = %v, want only ProtocolConfig", protocol, yamlNodeMapKeys(cfg))
			}
			if _, ok := cfg["ProtocolConfig"]; !ok {
				t.Fatalf("installed %s defaults have no ProtocolConfig", protocol)
			}
		})
	}
}

func yamlNodeMapKeys(m map[string]yaml.Node) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
