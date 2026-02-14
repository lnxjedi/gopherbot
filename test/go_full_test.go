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
		{aliceID, general, ";go-prompts", false, []TestMessage{
			{alice, general, "Codename check: pick a mission codename\\.", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 150},
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
		{aliceID, general, ";go-memory-seed", false, []TestMessage{
			{null, general, "MEMORY SEED: done", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-memory-check", false, []TestMessage{
			{null, general, "MEMORY CHECK: local=saffron noodles shared=solar soup ctx=orbital-7 thread=<empty> threadctx=<empty>", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, general, ";go-memory-check", false, []TestMessage{
			{null, general, "MEMORY CHECK: local=<empty> shared=solar soup ctx=<empty> thread=<empty> threadctx=<empty>", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-memory-thread-check", true, []TestMessage{
			{null, general, "MEMORY THREAD CHECK: local=<empty> shared=<empty> ctx=<empty> thread=delta thread threadctx=aurora mission", true}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-memory-datum-seed", false, []TestMessage{
			{null, general, "MEMORY DATUM SEED: update=Ok", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-memory-datum-check", false, []TestMessage{
			{null, general, "MEMORY DATUM CHECK: mission=opal-orbit vehicle=heron-7 status=go", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-memory-datum-checkin", false, []TestMessage{
			{null, general, "MEMORY DATUM CHECKIN: exists=true token=true ret=Ok", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-identity", false, []TestMessage{
			{null, general, "IDENTITY CHECK: bot=bender/Ok sender=Alice/Ok bob=Robert/Ok set=true param=<empty>", false}},
			[]Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";go-parameter-addtask", false, []TestMessage{
			{null, general, "SETPARAM ADDTASK: queued", false},
			{null, general, "PARAM-SHOW: PIPELINE_SENTINEL=nebula-42", false}},
			[]Event{CommandTaskRan, GoPluginRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestGoFullPipelineAdmin(t *testing.T) {
	if !wantFull("go") {
		t.Skip("skipping Go full pipeline/admin test; set RUN_FULL=go (or RUN_GOFULL=1)")
	}
	done, conn := setup("test/gofull", "/tmp/bottest.log", t)

	flow := []testItem{
		{aliceID, general, ";go-admin-check", false, []TestMessage{
			{null, general, "ADMIN CHECK: true", false}}, nil, 0},
		{bobID, general, ";go-admin-check", false, []TestMessage{
			{null, general, "ADMIN CHECK: false", false}}, nil, 0},
		{aliceID, general, ";go-elevate-check", false, []TestMessage{
			{alice, general, "This command requires immediate elevation.*TOTP code.*", false}}, nil, 150},
		{aliceID, general, "123456", false, []TestMessage{
			{null, general, "There were technical issues validating your code.*", false},
			{null, general, "Sorry, elevation failed due to a problem with the elevation service", false},
			{null, general, "ELEVATE CHECK: false", false}}, nil, 0},
		{aliceID, general, ";go-pipeline-ok", false, []TestMessage{
			{null, general, "PIPELINE OK: queued", false},
			{null, general, "PIPE NOTE: add-task", false},
			{null, general, "Starting job 'pipe-job job-step', run [0-9]+", false},
			{null, general, "PIPE NOTE: job-step", false},
			{null, general, "Finished job 'pipe-job', run [0-9]+, final task 'pipe-job', status: normal", false},
			{null, general, "PIPE ADD COMMAND: ran", false},
			{null, general, "PIPE FINAL COMMAND: ran", false},
			{null, general, "PIPE NOTE: final-task", false}}, nil, 0},
		{aliceID, general, ";go-pipeline-fail", false, []TestMessage{
			{null, general, "PIPELINE FAIL: armed", false},
			{null, general, "PIPE NOTE: fail-task", false},
			{null, general, "PIPE FAIL COMMAND: ran", false}}, nil, 0},
		{aliceID, general, ";go-spawn-job", false, []TestMessage{
			{null, general, "Starting job 'pipe-spawn-job spawn-step', run [0-9]+ - spawned by pipeline .*", false},
			{null, general, "PIPE NOTE: spawn-step", false},
			{null, general, "Finished job 'pipe-spawn-job', run [0-9]+, final task 'pipe-spawn-job', status: normal", false}}, nil, 0},
	}

	for _, step := range flow {
		testcaseRepliesOnly(t, conn, step)
	}

	teardown(t, done, conn)
}
