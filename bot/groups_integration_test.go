// +build integration

package bot_test

// lists_integration_test.go - verification of the 'lists' plugin functionality.

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/bot"
	testc "github.com/lnxjedi/gopherbot/connectors/test"
)

func TestGroupAuth(t *testing.T) {
	done, conn := setup("resources/cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, general, ";list groups", []testc.TestMessage{{null, general, "Here are the groups.*"}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckPassed}, 0},
		{alice, general, ";show the foo group", []testc.TestMessage{{null, general, "I don't have a .*"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{alice, general, ";show the Helpdesk group", []testc.TestMessage{{null, general, "(?m:The Helpdesk group has the following members:\ncarol\nbob$)"}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckPassed}, 0},
		{alice, general, ";remember The Alamo", []testc.TestMessage{{null, general, "Sorry, you're not authorized for .*"}}, []Event{GoPluginRan, AdminCheckPassed, AuthRanFail}, 0},
		{carol, general, ";list groups", []testc.TestMessage{{null, general, "(?m:Here are the groups you're an administrator for:\n.*\n.*$)"}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckFailed}, 0},
		{bob, general, ";remember Lieutenant Dan", []testc.TestMessage{{null, general, "Ok, .*"}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandTaskRan, ExternalTaskRan}, 0},
		{david, general, ";remember Forest Gump", []testc.TestMessage{{null, general, "Sorry, you're not authorized for .*"}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanFail}, 0},
		{carol, general, ";remember Forest Gump", []testc.TestMessage{{null, general, "Ok, I'll remember .*"}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandTaskRan, ExternalTaskRan}, 0},
		{carol, general, ";add david to the Helpdesk group", []testc.TestMessage{{null, general, "Ok, I added david to the Helpdesk group"}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckFailed}, 0},
		{alice, general, ";remove bob from the Helpdesk group", []testc.TestMessage{{null, general, "bob isn't a dynamic member of the Helpdesk group"}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckPassed}, 0},
		{david, general, ";remember Jenny", []testc.TestMessage{{null, general, "Ok, I'll remember .*"}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
