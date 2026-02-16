package bot

import "testing"

func TestTermBotHiddenPayload(t *testing.T) {
	payload, ok := termBotHiddenPayload("/floyd ping", "floyd")
	if !ok || payload != "floyd ping" {
		t.Fatalf("termBotHiddenPayload() basic = (%q, %t)", payload, ok)
	}

	payload, ok = termBotHiddenPayload("/floyd: ping", "floyd")
	if !ok || payload != "floyd ping" {
		t.Fatalf("termBotHiddenPayload() colon = (%q, %t)", payload, ok)
	}

	payload, ok = termBotHiddenPayload("/floyd, ping", "floyd")
	if !ok || payload != "floyd ping" {
		t.Fatalf("termBotHiddenPayload() comma = (%q, %t)", payload, ok)
	}

	if _, ok = termBotHiddenPayload("/ping", "floyd"); ok {
		t.Fatalf("termBotHiddenPayload() expected bare /ping to be rejected")
	}
}
