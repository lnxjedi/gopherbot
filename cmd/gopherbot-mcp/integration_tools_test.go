package main

import "testing"

func TestParseIntegrationSuiteListJSON(t *testing.T) {
	got, err := parseIntegrationSuiteList(`[{"name":"TestLuaFull","config_dir":"test/luafull","metadata":{"subsystems":["extension-api"],"runtimes":["lua"],"tier":"full"}}]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0]["name"] != "TestLuaFull" {
		t.Fatalf("name = %#v", got[0]["name"])
	}
	metadata, ok := got[0]["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata = %#v", got[0]["metadata"])
	}
	if metadata["tier"] != "full" {
		t.Fatalf("tier = %#v", metadata["tier"])
	}
}

func TestParseIntegrationSuiteListTSV(t *testing.T) {
	got, err := parseIntegrationSuiteList("TestBotName\ttest/membrain\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0]["config_dir"] != "test/membrain" {
		t.Fatalf("config_dir = %#v", got[0]["config_dir"])
	}
}
