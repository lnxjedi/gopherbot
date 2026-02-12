package bot

import "testing"

func TestResolveTermUserByNameAndID(t *testing.T) {
	tc := &termConnector{
		Handler:         handler{},
		users:           []termUser{{Name: "alice", InternalID: "u0001"}},
		userNameToIndex: map[string]int{"alice": 0},
		userIDToIndex:   map[string]int{"u0001": 0},
	}

	user, ok := tc.resolveTermUser("alice")
	if !ok {
		t.Fatalf("resolveTermUser by name failed")
	}
	if user.Name != "alice" || user.InternalID != "u0001" {
		t.Fatalf("resolveTermUser by name returned wrong user: %#v", user)
	}

	user, ok = tc.resolveTermUser("<u0001>")
	if !ok {
		t.Fatalf("resolveTermUser by ID failed")
	}
	if user.Name != "alice" || user.InternalID != "u0001" {
		t.Fatalf("resolveTermUser by ID returned wrong user: %#v", user)
	}
}

func TestResolveTermUserOutOfRangeIndex(t *testing.T) {
	tc := &termConnector{
		Handler:         handler{},
		users:           []termUser{{Name: "alice", InternalID: "u0001"}},
		userNameToIndex: map[string]int{"alice": 99},
		userIDToIndex:   map[string]int{"u0001": 99},
	}

	if _, ok := tc.resolveTermUser("alice"); ok {
		t.Fatalf("resolveTermUser should fail for out-of-range name index")
	}
	if _, ok := tc.resolveTermUser("<u0001>"); ok {
		t.Fatalf("resolveTermUser should fail for out-of-range ID index")
	}
	if _, ok := tc.resolveTermUserByName("alice"); ok {
		t.Fatalf("resolveTermUserByName should fail for out-of-range index")
	}
	if _, ok := tc.resolveTermUserByID("u0001"); ok {
		t.Fatalf("resolveTermUserByID should fail for out-of-range index")
	}
}
