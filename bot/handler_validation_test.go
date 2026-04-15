package bot

import "testing"

func TestResolveIncomingUserKeepsConnectorUserButRequiresValidationSeparately(t *testing.T) {
	maps := &userChanMaps{
		userIDProto: map[string]map[string]*UserInfo{},
		directoryUser: map[string]bool{
			"alice": true,
		},
		user: map[string]*DirectoryUser{
			"alice": {UserName: "alice"},
		},
	}

	name, _, protocolMapped, directoryListed := resolveIncomingUser(maps, "slack", "U001", "alice")

	if name != "alice" {
		t.Fatalf("resolveIncomingUser() name=%q, want %q", name, "alice")
	}
	if !protocolMapped {
		t.Fatal("resolveIncomingUser() protocolMapped=false, want true because connector supplied username")
	}
	if !directoryListed {
		t.Fatal("resolveIncomingUser() directoryListed=false, want true")
	}
	if shouldAcceptIncomingUser(directoryListed, false, false) {
		t.Fatal("unvalidated directory user should be rejected when IgnoreUnlistedUsers is false")
	}
}
