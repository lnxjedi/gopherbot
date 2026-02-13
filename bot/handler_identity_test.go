package bot

import "testing"

func TestResolveIncomingUserProtocolAndDirectory(t *testing.T) {
	alice := &UserInfo{UserName: "alice", UserID: "U001"}
	maps := &userChanMaps{
		userIDProto: map[string]map[string]*UserInfo{"slack": {"U001": alice}},
		directoryUser: map[string]bool{
			"alice": true,
		},
	}

	name, botUser, protocolMapped, directoryListed := resolveIncomingUser(maps, "slack", "U001", "Alice")

	if name != "alice" {
		t.Fatalf("resolveIncomingUser() name=%q, want %q", name, "alice")
	}
	if botUser {
		t.Fatal("resolveIncomingUser() botUser=true, want false")
	}
	if !protocolMapped {
		t.Fatal("resolveIncomingUser() protocolMapped=false, want true")
	}
	if !directoryListed {
		t.Fatal("resolveIncomingUser() directoryListed=false, want true")
	}
}

func TestResolveIncomingUserProtocolMapWithoutDirectory(t *testing.T) {
	bob := &UserInfo{UserName: "bob", UserID: "U002"}
	maps := &userChanMaps{
		userIDProto: map[string]map[string]*UserInfo{"slack": {"U002": bob}},
		directoryUser: map[string]bool{
			"alice": true,
		},
	}

	name, _, protocolMapped, directoryListed := resolveIncomingUser(maps, "slack", "U002", "Bob")

	if name != "bob" {
		t.Fatalf("resolveIncomingUser() name=%q, want %q", name, "bob")
	}
	if !protocolMapped {
		t.Fatal("resolveIncomingUser() protocolMapped=false, want true")
	}
	if directoryListed {
		t.Fatal("resolveIncomingUser() directoryListed=true, want false")
	}
}

func TestResolveIncomingUserNoProtocolMapNotListed(t *testing.T) {
	maps := &userChanMaps{
		userIDProto: map[string]map[string]*UserInfo{},
		directoryUser: map[string]bool{
			"alice": true,
		},
	}

	name, _, protocolMapped, directoryListed := resolveIncomingUser(maps, "slack", "U001", "alice")

	if name != "alice" {
		t.Fatalf("resolveIncomingUser() name=%q, want %q", name, "alice")
	}
	if protocolMapped {
		t.Fatal("resolveIncomingUser() protocolMapped=true, want false")
	}
	if !directoryListed {
		t.Fatal("resolveIncomingUser() directoryListed=false, want true")
	}
}

func TestResolveIncomingUserUnmappedFallsBackToConnectorOrBracket(t *testing.T) {
	maps := &userChanMaps{
		userIDProto:   map[string]map[string]*UserInfo{},
		directoryUser: map[string]bool{},
	}

	name, _, protocolMapped, directoryListed := resolveIncomingUser(maps, "slack", "U404", "")

	if name != bracket("U404") {
		t.Fatalf("resolveIncomingUser() name=%q, want %q", name, bracket("U404"))
	}
	if protocolMapped {
		t.Fatal("resolveIncomingUser() protocolMapped=true, want false")
	}
	if directoryListed {
		t.Fatal("resolveIncomingUser() directoryListed=true, want false")
	}
}
