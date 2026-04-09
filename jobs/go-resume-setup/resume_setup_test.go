package main

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestJoinedUserParsesJoinMessage(t *testing.T) {
	msg := &robot.Message{
		Incoming: &robot.ConnectorMessage{
			MessageText: "@alice has joined #general",
		},
	}
	user, channel := joinedUser(nil, msg)
	if user != "alice" || channel != "general" {
		t.Fatalf("joinedUser() = (%q, %q), want (%q, %q)", user, channel, "alice", "general")
	}
}

func TestJoinedUserUsesArgsWhenProvided(t *testing.T) {
	user, channel := joinedUser([]string{"Samantha", "General"}, nil)
	if user != "samantha" || channel != "general" {
		t.Fatalf("joinedUser(args) = (%q, %q), want (%q, %q)", user, channel, "samantha", "general")
	}
}

func TestJoinedUserFromLookupDoesNotCallLookupWhenArgsProvided(t *testing.T) {
	user, channel := joinedUserFromLookup([]string{"Samantha", "General"}, func() *robot.Message {
		t.Fatal("joinedUserFromLookup unexpectedly called lookup")
		return nil
	})
	if user != "samantha" || channel != "general" {
		t.Fatalf("joinedUserFromLookup(args) = (%q, %q), want (%q, %q)", user, channel, "samantha", "general")
	}
}
