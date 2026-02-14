//go:build integration
// +build integration

package tbot_test

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
)

func TestGoFull(t *testing.T) {
	if !wantFull("go") {
		t.Skip("skipping Go full test; set RUN_FULL=go (or RUN_GOFULL=1)")
	}
	done, conn := setup("test/gofull", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, ";say everything", false, []TestMessage{
			{null, general, "Regular Say", false},
			{null, general, "SayThread, yeah", true},
			{alice, general, "Regular Reply", false},
			{alice, general, "Reply in thread, yo", true},
			{null, general, "Sending to the channel: general", false},
			{alice, null, "Sending this message to user: alice", false},
			{alice, general, "Sending to user 'alice' in channel: general", false},
			{null, general, "Sending to channel 'general' in thread: 0xDEADBEEF", true},
			{alice, general, "Sending to user 'alice' in channel 'general' in thread: 0xDEADBEEF", true}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/;say everything", false, []TestMessage{
			{null, general, "(Regular Say)", false},
			{null, general, "(SayThread, yeah)", true},
			{alice, general, "(Regular Reply)", false},
			{alice, general, "(Reply in thread, yo)", true},
			{null, general, "(Sending to the channel: general)", false},
			{alice, null, "(Sending this message to user: alice)", false},
			{alice, general, "(Sending to user 'alice' in channel: general)", false},
			{null, general, "(Sending to channel 'general' in thread: 0xDEADBEEF)", true},
			{alice, general, "(Sending to user 'alice' in channel 'general' in thread: 0xDEADBEEF)", true}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-config", false, []TestMessage{
			{null, general, "Not completely random.*", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-subscribe", false, []TestMessage{
			{null, general, "SUBSCRIBE FLOW: true/true", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
