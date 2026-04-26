//go:build integration
// +build integration

package tbot_test

import (
	"strings"
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
	testc "github.com/lnxjedi/gopherbot/v2/connectors/test"
)

func TestHiddenAdminInspectCommands(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest-admin-hidden.log", t)

	tests := []testItem{
		{aliceID, null, "dump robot", false, []TestMessage{{alice, null, "This command is only available as a hidden command.", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: dump robot", false, []TestMessage{{null, general, "HERE'S HOW I'VE BEEN CONFIGURED.*", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin echo", false, []TestMessage{{null, general, "ALLCHANNELS.*", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin default echo", false, []TestMessage{{null, general, "HERE'S.*", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin junk", false, []TestMessage{{null, general, "Didn't find .* junk", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	WaitForBackgroundInitsForTesting()
	GetEvents()
	conn.SendBotMessage(&testc.TestMessage{aliceID, general, "bender: list plugins", false, true})
	got, err := conn.GetBotMessage()
	if err != nil {
		t.Fatalf("timed out waiting for list plugins reply: %v", err)
	}
	if !strings.Contains(got.Message, "builtin-admin") {
		t.Fatalf("list plugins missing builtin-admin: %q", got.Message)
	}
	if strings.Contains(got.Message, "builtin-dmadmin") {
		t.Fatalf("list plugins still includes builtin-dmadmin: %q", got.Message)
	}
	ev := GetEvents()
	want := []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}
	if len(*ev) != len(want) {
		t.Fatalf("list plugins events = %v, want %v", *ev, want)
	}
	for i, event := range *ev {
		if event != want[i] {
			t.Fatalf("list plugins events = %v, want %v", *ev, want)
		}
	}

	teardown(t, done, conn)
}
