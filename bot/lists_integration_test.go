// +build integration

package bot_test

// lists_integration_test.go - verification of the 'lists' plugin functionality.

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/bot"
	testc "github.com/lnxjedi/gopherbot/connectors/test"
)

func TestLists(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

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
		{alice, general, ";list lists", []testc.TestMessage{{null, general, "I don't have any lists"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{alice, general, ";add burgers to the meals list", []testc.TestMessage{{alice, general, "I don't have a .*"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bob, random, ";add burgers to the meals list", []testc.TestMessage{{bob, random, "I don't have a .*"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bob, random, "yes", []testc.TestMessage{{null, random, "Ok, I created a new meals list and added burgers to it"}}, []Event{}, 0},
		{alice, general, "yes", []testc.TestMessage{{null, general, "Somebody already created the meals list and added burgers to it"}}, []Event{}, 0},
		{alice, general, ";add eggs to the Breakfast list", []testc.TestMessage{{alice, general, "I don't have .*"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bob, random, ";add bacon to the breakfast list", []testc.TestMessage{{bob, random, "I don't have a .*"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bob, random, "yes", []testc.TestMessage{{null, random, "Ok, I created a new breakfast list and added bacon to it"}}, []Event{}, 0},
		{alice, general, "yes", []testc.TestMessage{{null, general, "Ok, I added eggs to the new Breakfast list"}}, []Event{}, 0},
		{alice, general, ";show the meals list", []testc.TestMessage{{null, general, `(?m:Here's what I have.*\nburgers$)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{carol, general, ";remove BURGERS from the meals list", []testc.TestMessage{{null, general, "Ok, I removed BURGERS from the meals list"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{carol, general, ";show the list", []testc.TestMessage{{null, general, "The meals list is empty"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{carol, general, ";delete the breakFAST list", []testc.TestMessage{{null, general, "Deleted"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{alice, general, ";list lists", []testc.TestMessage{{null, general, `(?m:Here are the lists I know about:\nmeals)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{alice, general, ";show the MEALS list", []testc.TestMessage{{null, general, "The MEALS list is empty"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
