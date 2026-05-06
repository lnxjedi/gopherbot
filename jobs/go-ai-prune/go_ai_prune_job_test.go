package main

import (
	"testing"
	"time"
)

func TestSelectStaleConversationsFiltersAndSorts(t *testing.T) {
	cutoff := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	idx := conversationIndex{
		Conversations: map[string]conversationIndexEntry{
			"thread:new": {
				Key:       "k-new",
				UpdatedAt: "2026-03-01T12:00:00Z",
			},
			"thread:old-b": {
				Key:       "k-old-b",
				UpdatedAt: "2026-02-01T00:00:00Z",
			},
			"thread:old-a": {
				Key:       "k-old-a",
				UpdatedAt: "2026-01-01T00:00:00Z",
			},
			"thread:invalid-time": {
				Key:       "k-invalid",
				UpdatedAt: "not-a-time",
			},
			"thread:missing-key": {
				Key:       "",
				UpdatedAt: "2026-01-01T00:00:00Z",
			},
		},
	}

	stale := selectStaleConversations(idx, cutoff)
	if len(stale) != 2 {
		t.Fatalf("stale size = %d, want 2", len(stale))
	}
	if stale[0].ID != "thread:old-a" {
		t.Fatalf("stale[0].id = %q, want %q", stale[0].ID, "thread:old-a")
	}
	if stale[1].ID != "thread:old-b" {
		t.Fatalf("stale[1].id = %q, want %q", stale[1].ID, "thread:old-b")
	}
}

func TestLimitStaleConversations(t *testing.T) {
	in := []staleConversation{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}
	limited := limitStaleConversations(in, 2)
	if len(limited) != 2 {
		t.Fatalf("limited size = %d, want 2", len(limited))
	}
	if limited[0].ID != "a" || limited[1].ID != "b" {
		t.Fatalf("unexpected limited order/content: %#v", limited)
	}
	unlimited := limitStaleConversations(in, 0)
	if len(unlimited) != 3 {
		t.Fatalf("unlimited size = %d, want 3", len(unlimited))
	}
}

func TestRemoveDeletedFromIndexHonorsExpectedKey(t *testing.T) {
	idx := conversationIndex{
		Conversations: map[string]conversationIndexEntry{
			"dm:alice": {Key: "k1", UpdatedAt: "2026-01-01T00:00:00Z"},
			"dm:bob":   {Key: "k2", UpdatedAt: "2026-01-01T00:00:00Z"},
		},
	}
	deleted := map[string]string{
		"dm:alice": "k1",
		"dm:bob":   "wrong",
		"dm:carol": "k3",
	}
	removed := removeDeletedFromIndex(&idx, deleted)
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, ok := idx.Conversations["dm:alice"]; ok {
		t.Fatal("expected dm:alice to be removed")
	}
	if _, ok := idx.Conversations["dm:bob"]; !ok {
		t.Fatal("expected dm:bob to remain")
	}
}
