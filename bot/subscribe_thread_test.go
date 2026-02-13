package bot

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSubscriptionsUnmarshalLegacyKeyFormat(t *testing.T) {
	var s tSubs
	data := []byte(`{"general{|}0005":{"Plugin":"plug","Timestamp":"2026-02-13T00:00:00Z"}}`)
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("json.Unmarshal legacy subscriptions failed: %v", err)
	}
	key := subscriptionMatcher{channel: "general", thread: "0005"}
	sub, ok := s.m[key]
	if !ok {
		t.Fatalf("legacy key not found after unmarshal: %#v", s.m)
	}
	if sub.Plugin != "plug" {
		t.Fatalf("legacy subscription plugin = %q, want %q", sub.Plugin, "plug")
	}
}

func TestSubscriptionsMarshalProtocolKeyFormat(t *testing.T) {
	s := tSubs{
		m: map[subscriptionMatcher]subscriber{
			{protocol: "slack", channel: "general", thread: "0005"}: {
				Plugin:    "plug",
				Timestamp: time.Unix(0, 0).UTC(),
			},
		},
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal subscriptions failed: %v", err)
	}
	var got map[string]subscriber
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal marshalled subscriptions failed: %v", err)
	}
	if _, ok := got["slack{|}general{|}0005"]; !ok {
		t.Fatalf("expected protocol-scoped key in marshalled data, got keys: %#v", got)
	}
}

func TestLookupSubscriptionLockedPrefersProtocol(t *testing.T) {
	subscriptions.Lock()
	origMap := subscriptions.m
	origDirty := subscriptions.dirty
	subscriptions.m = map[subscriptionMatcher]subscriber{
		{channel: "general", thread: "0005"}: {
			Plugin: "legacy-plug",
		},
		{protocol: "slack", channel: "general", thread: "0005"}: {
			Plugin: "slack-plug",
		},
	}
	subscriptions.dirty = false
	subscriptions.Unlock()
	t.Cleanup(func() {
		subscriptions.Lock()
		subscriptions.m = origMap
		subscriptions.dirty = origDirty
		subscriptions.Unlock()
	})

	subscriptions.Lock()
	key, sub, ok := lookupSubscriptionLocked("slack", "general", "0005")
	subscriptions.Unlock()
	if !ok {
		t.Fatal("lookupSubscriptionLocked(slack) returned not found")
	}
	if key.protocol != "slack" || sub.Plugin != "slack-plug" {
		t.Fatalf("lookupSubscriptionLocked(slack) = key:%#v sub:%#v, want protocol match", key, sub)
	}

	subscriptions.Lock()
	key, sub, ok = lookupSubscriptionLocked("ssh", "general", "0005")
	subscriptions.Unlock()
	if !ok {
		t.Fatal("lookupSubscriptionLocked(ssh) returned not found")
	}
	if key.protocol != "" || sub.Plugin != "legacy-plug" {
		t.Fatalf("lookupSubscriptionLocked(ssh) = key:%#v sub:%#v, want legacy fallback", key, sub)
	}
}
