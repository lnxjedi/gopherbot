// +build integration

package bot_test

// lists_integration_test.go - verification of the 'lists' plugin functionality.

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/bot"
	testc "github.com/lnxjedi/gopherbot/connectors/test"
)

func TestGroupAuth(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottestlists.log", t)

	tests := []testItem{
		{alice, general, ";list groups", []testc.TestMessage{{null, general, "Here are the groups.*"}}, []Event{CommandPluginRan, GoPluginRan, AdminCheckPassed}, 0},
		{alice, general, ";show the foo group", []testc.TestMessage{{null, general, "I don't have a .*"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, ";show the Helpdesk group", []testc.TestMessage{{null, general, "(?m:The Helpdesk group has the following members:\ncarol\nbob$)"}}, []Event{CommandPluginRan, GoPluginRan, AdminCheckPassed}, 0},
		{alice, general, ";remember The Alamo", []testc.TestMessage{{null, general, "Sorry, you're not authorized for .*"}}, []Event{GoPluginRan, AdminCheckPassed, AuthRanFail}, 0},
		{carol, general, ";list groups", []testc.TestMessage{{null, general, "(?m:Here are the groups you're an administrator for:\n.*\n.*$)"}}, []Event{CommandPluginRan, GoPluginRan, AdminCheckFailed}, 0},
		{bob, general, ";remember Lieutenant Dan", []testc.TestMessage{{null, general, "Ok, .*"}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandPluginRan, ScriptPluginRan}, 0},
		{david, general, ";remember Forest Gump", []testc.TestMessage{{null, general, "Sorry, you're not authorized for .*"}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanFail}, 0},
		{carol, general, ";remember Forest Gump", []testc.TestMessage{{null, general, "Ok, I'll remember .*"}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandPluginRan, ScriptPluginRan}, 0},
		{carol, general, ";add david to the Helpdesk group", []testc.TestMessage{{null, general, "Ok, I added david to the Helpdesk group"}}, []Event{CommandPluginRan, GoPluginRan, AdminCheckFailed}, 0},
		{david, general, ";remember Jenny", []testc.TestMessage{{null, general, "Ok, I'll remember .*"}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandPluginRan, ScriptPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
