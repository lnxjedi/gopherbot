package bot

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEphemeralMemoriesUnmarshalLegacyContextKey(t *testing.T) {
	src := map[string]ephemeralMemory{
		"context:thing{|}parsley{|}general{|}0001": {
			Memory:    "value",
			Timestamp: time.Now().UTC(),
		},
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var memories eMemories
	if err := json.Unmarshal(data, &memories); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	ctx := memoryContext{
		key:      "context:thing",
		user:     "parsley",
		channel:  "general",
		thread:   "0001",
		protocol: "",
	}
	if _, ok := memories.m[ctx]; !ok {
		t.Fatalf("legacy context key was not restored: %+v", ctx)
	}
}

func TestEphemeralMemoriesMarshalIncludesProtocolField(t *testing.T) {
	memories := eMemories{
		m: map[memoryContext]ephemeralMemory{
			{
				key:      "context:thing",
				user:     "parsley",
				channel:  "general",
				thread:   "0001",
				protocol: "ssh",
			}: {
				Memory:    "value",
				Timestamp: time.Now().UTC(),
			},
		},
	}

	data, err := json.Marshal(memories)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	var out map[string]ephemeralMemory
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if _, ok := out["context:thing{|}parsley{|}general{|}0001{|}ssh"]; !ok {
		t.Fatalf("serialized key missing protocol segment: %#v", out)
	}
}
