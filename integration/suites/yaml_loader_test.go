package suites

import "testing"

func TestYAMLSuitesLoaded(t *testing.T) {
	suites := List()
	if len(suites) != 45 {
		t.Fatalf("suite count = %d, want 45", len(suites))
	}
	for _, suite := range suites {
		if suite.Name == "" {
			t.Fatal("loaded suite with empty name")
		}
		if suite.ConfigDir == "" {
			t.Fatalf("%s: empty config dir", suite.Name)
		}
		if len(suite.Cases) == 0 && suite.Flow == nil {
			t.Fatalf("%s: no cases or flow", suite.Name)
		}
		if len(suite.Metadata.Subsystems) == 0 {
			t.Fatalf("%s: no metadata subsystems", suite.Name)
		}
		if suite.Metadata.Tier == "" {
			t.Fatalf("%s: no metadata tier", suite.Name)
		}
	}
}

func TestYAMLInputUserNamesResolveToConnectorIDs(t *testing.T) {
	c, err := yamlCaseToCase(yamlCase{
		Input: yamlMessage{
			User: Alice,
			Text: "ping",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Input.User != AliceID {
		t.Fatalf("input user = %q, want %q", c.Input.User, AliceID)
	}
}
