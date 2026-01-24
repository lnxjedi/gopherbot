//go:build integration
// +build integration

package tbot_test

import (
	"os"
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
)

func TestLuaFull(t *testing.T) {
	if !wantFull("lua") {
		t.Skip("skipping Lua full test; set RUN_FULL=lua (or RUN_LUAFULL=1)")
	}
	baseURL, closeServer := startTestHTTPServer(t)
	defer closeServer()
	os.Setenv("GBOT_TEST_HTTP_BASEURL", baseURL)
	defer os.Unsetenv("GBOT_TEST_HTTP_BASEURL")
	done, conn := setup("test/luafull", "/tmp/bottest.log", t)

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
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
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
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-config", false, []TestMessage{
			{null, general, "Not completely random.*", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-http", false, []TestMessage{
			{null, general, "HTTP GET ok: GET", false},
			{null, general, "HTTP POST ok: alpha", false},
			{null, general, "HTTP PUT ok: bravo", false},
			{null, general, "HTTP ERROR ok: 500", false},
			{null, general, "HTTP TIMEOUT ok", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
