//go:build integration
// +build integration

package tbot_test

// lists_integration_test.go - verification of the 'lists' plugin functionality.

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
)

func TestLists(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	/* Note on ordering:

	Be careful with the plugins you're testing, and be sure that the robot
	completes all actions before replying. Consider for instance:

		Say "I'll remember \"$1\" is \"$2\" - but eventually I'll forget!"
		Remember "$1" "$2"

	This order of events means the test may well complete (because it got the
	reply) before actually remembering the fact. The next test, recalling the
	fact, could then fail because it tries to recall the fact before it's
	actually been stored in the previous test.

	I know this because it took me a couple of hours to figure out why my
	test was failing. */

	tests := []testItem{
		{aliceID, general, ";list lists", false, []TestMessage{{null, general, "I don't have any lists", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";add burgers to the meals list", false, []TestMessage{{alice, general, "I don't have a .*", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, random, ";add burgers to the meals list", false, []TestMessage{{bob, random, "I don't have a .*", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, random, "yes", false, []TestMessage{{null, random, "Ok, I created a new meals list and added burgers to it", false}}, []Event{}, 0},
		{aliceID, general, "yes", false, []TestMessage{{null, general, "Somebody already created the meals list and added burgers to it", false}}, []Event{}, 0},
		{aliceID, general, ";add eggs to the Breakfast list", false, []TestMessage{{alice, general, "I don't have .*", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, random, ";add bacon to the breakfast list", false, []TestMessage{{bob, random, "I don't have a .*", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, random, "yes", false, []TestMessage{{null, random, "Ok, I created a new breakfast list and added bacon to it", false}}, []Event{}, 0},
		{aliceID, general, "yes", false, []TestMessage{{null, general, "Ok, I added eggs to the new Breakfast list", false}}, []Event{}, 0},
		{aliceID, general, ";show the meals list", false, []TestMessage{{null, general, `(?m:Here's what I have.*\nburgers$)`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{carolID, general, ";remove BURGERS from the meals list", false, []TestMessage{{null, general, "Ok, I removed BURGERS from the meals list", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{carolID, general, ";show the list", false, []TestMessage{{null, general, "The meals list is empty", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{carolID, general, ";delete the breakFAST list", false, []TestMessage{{null, general, "Deleted", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";list lists", false, []TestMessage{{null, general, `(?m:Here are the lists I know about:\nmeals)`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";show the MEALS list", false, []TestMessage{{null, general, "The MEALS list is empty", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
