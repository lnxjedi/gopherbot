package main

import (
	"testing"
	"time"
)

func TestParseReplyCommandContextUsesCommandRobotDir(t *testing.T) {
	ctx, err := parseReplyCommandContext(map[string]interface{}{
		"text": "hello",
		"command": map[string]interface{}{
			"robot_dir": "/tmp/robot-a",
			"protocol":  "ssh",
			"channel":   "general",
			"thread_id": "t-1",
			"user_name": "david",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.RobotDir != "/tmp/robot-a" {
		t.Fatalf("robot dir=%q, want /tmp/robot-a", ctx.RobotDir)
	}
	if ctx.Direct {
		t.Fatal("expected non-direct context")
	}
}

func TestParseReplyCommandContextDirectRequiresUser(t *testing.T) {
	_, err := parseReplyCommandContext(map[string]interface{}{
		"text":   "hello",
		"direct": true,
		"command": map[string]interface{}{
			"robot_dir": "/tmp/robot-a",
			"protocol":  "ssh",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestParseReplyCommandContextRequiresChannelWhenNonDirect(t *testing.T) {
	_, err := parseReplyCommandContext(map[string]interface{}{
		"text": "hello",
		"command": map[string]interface{}{
			"robot_dir": "/tmp/robot-a",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCommandsWithRobotAddsRobotDir(t *testing.T) {
	res := map[string]interface{}{
		"commands": []interface{}{
			map[string]interface{}{
				"command": "hi",
			},
		},
	}
	out := commandsWithRobot(res, "/tmp/r1")
	if len(out) != 1 {
		t.Fatalf("len(out)=%d, want 1", len(out))
	}
	cmd, ok := out[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected map command")
	}
	if got := cmd["robot_dir"]; got != "/tmp/r1" {
		t.Fatalf("robot_dir=%v, want /tmp/r1", got)
	}
}

func TestFirstCommandTimestampParsesRFC3339Nano(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res := map[string]interface{}{
		"commands": []interface{}{
			map[string]interface{}{"timestamp": now},
		},
	}
	got := firstCommandTimestamp(res)
	if got.IsZero() {
		t.Fatal("expected parsed timestamp")
	}
}

func TestShouldResyncAIDevCursor(t *testing.T) {
	tests := []struct {
		name          string
		streamLatest  uint64
		sessionCursor uint64
		want          bool
	}{
		{name: "no prior cursor", streamLatest: 0, sessionCursor: 0, want: false},
		{name: "normal forward progress", streamLatest: 7, sessionCursor: 3, want: false},
		{name: "equal cursor", streamLatest: 3, sessionCursor: 3, want: false},
		{name: "stream reset to zero", streamLatest: 0, sessionCursor: 3, want: true},
		{name: "stream reset to lower non-zero", streamLatest: 2, sessionCursor: 9, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldResyncAIDevCursor(tt.streamLatest, tt.sessionCursor)
			if got != tt.want {
				t.Fatalf("shouldResyncAIDevCursor(%d,%d)=%t want %t", tt.streamLatest, tt.sessionCursor, got, tt.want)
			}
		})
	}
}
