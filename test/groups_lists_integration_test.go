//go:build integration
// +build integration

package tbot_test

// groups_lists_integration_tests.go - verification of the 'lists' plugin functionality.

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
)

func TestGroupAuth(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, ";list groups", false, []TestMessage{{null, general, "Here are the groups.*", false}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckPassed}, 0},
		{aliceID, general, ";show the foo group", false, []TestMessage{{null, general, "I don't have a .*", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";show the Helpdesk group", false, []TestMessage{{null, general, "(?m:The Helpdesk group has the following members:\ncarol\nbob$)", false}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckPassed}, 0},
		{aliceID, general, ";remember The Alamo", false, []TestMessage{{null, general, "Sorry, you're not authorized for .*", false}}, []Event{GoPluginRan, AdminCheckPassed, AuthRanFail}, 0},
		{carolID, general, ";list groups", false, []TestMessage{{null, general, "(?m:Here are the groups you're an administrator for:\n.*\n.*$)", false}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckFailed}, 0},
		{bobID, general, ";remember Lieutenant Dan", false, []TestMessage{{null, general, "Ok, .*", false}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, general, ";remember Forest Gump", false, []TestMessage{{null, general, "Sorry, you're not authorized for .*", false}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanFail}, 0},
		{carolID, general, ";remember Forest Gump", false, []TestMessage{{null, general, "Ok, I'll remember .*", false}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandTaskRan, ExternalTaskRan}, 0},
		{carolID, general, ";add david to the Helpdesk group", false, []TestMessage{{null, general, "Ok, I added david to the Helpdesk group", false}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckFailed}, 0},
		{davidID, general, ";remember Jenny", false, []TestMessage{{null, general, "Ok, I'll remember .*", false}}, []Event{GoPluginRan, AdminCheckFailed, AuthRanSuccess, CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";remove bob from the Helpdesk group", false, []TestMessage{{null, general, "bob isn't a dynamic member of the Helpdesk group", false}}, []Event{CommandTaskRan, GoPluginRan, AdminCheckPassed}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
