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
		{aliceID, general, ";lua-subscribe", false, []TestMessage{
			{null, general, "SUBSCRIBE FLOW: true/true", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-prompts", false, []TestMessage{
			{alice, general, "Codename check: pick a mission codename\\.", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 150},
		{aliceID, general, "Nova Sparrow", false, []TestMessage{
			{alice, general, "Thread check: pick a favorite snack for launch\\.", true}},
			[]Event{}, 150},
		{aliceID, general, "spicy popcorn", true, []TestMessage{
			{alice, null, "DM check: name a secret moon base\\.", false}},
			[]Event{}, 150},
		{aliceID, null, "io station nine", false, []TestMessage{
			{alice, general, "Channel check: describe launch weather in two words\\.", false}},
			[]Event{BotDirectMessage}, 150},
		{aliceID, general, "aurora clear", false, []TestMessage{
			{alice, general, "Thread rally: choose a backup call sign\\.", true}},
			[]Event{}, 150},
		{aliceID, general, "ember fox", true, []TestMessage{
			{null, general, "PROMPT FLOW OK: Nova Sparrow \\| spicy popcorn \\| io station nine \\| aurora clear \\| ember fox", false}},
			[]Event{}, 0},
		{aliceID, general, ";lua-memory-seed", false, []TestMessage{
			{null, general, "MEMORY SEED: done", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-memory-check", false, []TestMessage{
			{null, general, "MEMORY CHECK: local=saffron noodles shared=solar soup ctx=orbital-7 thread=<empty> threadctx=<empty>", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{bobID, general, ";lua-memory-check", false, []TestMessage{
			{null, general, "MEMORY CHECK: local=<empty> shared=solar soup ctx=<empty> thread=<empty> threadctx=<empty>", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-memory-thread-check", true, []TestMessage{
			{null, general, "MEMORY THREAD CHECK: local=<empty> shared=<empty> ctx=<empty> thread=delta thread threadctx=aurora mission", true}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-memory-datum-seed", false, []TestMessage{
			{null, general, "MEMORY DATUM SEED: update=Ok", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-memory-datum-check", false, []TestMessage{
			{null, general, "MEMORY DATUM CHECK: mission=opal-orbit vehicle=heron-7 status=go", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-memory-datum-checkin", false, []TestMessage{
			{null, general, "MEMORY DATUM CHECKIN: exists=true token=true ret=Ok", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";lua-identity", false, []TestMessage{
			{null, general, "IDENTITY CHECK: bot=bender/Ok sender=Alice/Ok bob=Robert/Ok set=true param=<empty>", false}},
			[]Event{CommandTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
