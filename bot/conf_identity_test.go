package bot

import "testing"

func TestNormalizeUserMapForProtocol(t *testing.T) {
	in := map[string]string{
		"alice": "U001",
		"Bob":   "U002",
		"carol": "",
		"   ":   "U004",
		"david": " U005 ",
	}

	got := normalizeUserMapForProtocol("slack", in, "test")

	if len(got) != 2 {
		t.Fatalf("normalizeUserMapForProtocol() returned %d entries, want 2", len(got))
	}
	if got["alice"] != "U001" {
		t.Fatalf("normalizeUserMapForProtocol() alice = %q, want %q", got["alice"], "U001")
	}
	if got["david"] != "U005" {
		t.Fatalf("normalizeUserMapForProtocol() david = %q, want %q", got["david"], "U005")
	}
	if _, ok := got["Bob"]; ok {
		t.Fatal("normalizeUserMapForProtocol() should reject uppercase usernames")
	}
	if _, ok := got["carol"]; ok {
		t.Fatal("normalizeUserMapForProtocol() should reject empty user IDs")
	}
}

func TestExtractLegacyRosterIDs(t *testing.T) {
	raw := []byte(`[
		{"UserName":"alice","UserID":"U001"},
		{"UserName":"bob","UserID":"U002"},
		{"UserName":"carol","UserID":""},
		{"UserName":"david"},
		{"UserName":"","UserID":"U005"}
	]`)

	got := extractLegacyRosterIDs(raw)
	if len(got) != 2 {
		t.Fatalf("extractLegacyRosterIDs() returned %d entries, want 2", len(got))
	}
	if got["alice"] != "U001" || got["bob"] != "U002" {
		t.Fatalf("extractLegacyRosterIDs() = %#v, want alice/bob IDs", got)
	}
}

func TestMergeLegacyUserMapForProtocol(t *testing.T) {
	userMap := map[string]string{
		"alice": "NEW001",
	}
	legacy := map[string]string{
		"alice": "OLD001",
		"bob":   "OLD002",
	}

	used := mergeLegacyUserMapForProtocol("slack", userMap, legacy, "test")
	if used != 1 {
		t.Fatalf("mergeLegacyUserMapForProtocol() used=%d, want 1", used)
	}
	if userMap["alice"] != "NEW001" {
		t.Fatalf("mergeLegacyUserMapForProtocol() should keep explicit map value, got %q", userMap["alice"])
	}
	if userMap["bob"] != "OLD002" {
		t.Fatalf("mergeLegacyUserMapForProtocol() should fill missing legacy value, got %q", userMap["bob"])
	}
}
