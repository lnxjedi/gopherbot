package test

import "testing"

func cloneMap(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func TestRebuildUserIndexesReplacesOldEntries(t *testing.T) {
	oldIDMap := cloneMap(userIDMap)
	oldUserMap := cloneMap(userMap)
	defer func() {
		userIDMap = oldIDMap
		userMap = oldUserMap
	}()

	rebuildUserIndexes([]testUser{
		{Name: "alice", InternalID: "u0001"},
		{Name: "bob", InternalID: "u0002"},
	})
	if _, ok := userMap["alice"]; !ok {
		t.Fatalf("expected alice in userMap after first rebuild")
	}
	if _, ok := userIDMap["u0001"]; !ok {
		t.Fatalf("expected u0001 in userIDMap after first rebuild")
	}

	rebuildUserIndexes([]testUser{
		{Name: "carol", InternalID: "u0003"},
	})
	if _, ok := userMap["alice"]; ok {
		t.Fatalf("expected stale alice entry to be removed after second rebuild")
	}
	if _, ok := userIDMap["u0001"]; ok {
		t.Fatalf("expected stale u0001 entry to be removed after second rebuild")
	}
	if idx, ok := userMap["carol"]; !ok || idx != 0 {
		t.Fatalf("expected carol to be indexed at 0, got idx=%d ok=%t", idx, ok)
	}
	if idx, ok := userIDMap["u0003"]; !ok || idx != 0 {
		t.Fatalf("expected u0003 to be indexed at 0, got idx=%d ok=%t", idx, ok)
	}
}
