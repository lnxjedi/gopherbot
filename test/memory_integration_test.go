//go:build integration
// +build integration

package tbot_test

// memory_integration_test.go - tests that stress the robot's memory functions.

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
)

func TestMemory(t *testing.T) {
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
		{carolID, random, ";remember slowly The Alamo", false, []TestMessage{{null, random, "Ok, .*", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, random, ";remember Ferris Bueller", false, []TestMessage{{null, random, "Ok, .*", false}, {null, random, "committed to memory", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{bobID, random, "recall 1, Bender", false, []TestMessage{{null, random, "Ferris Bueller", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{carolID, random, ";remember Ferris Bueller", false, []TestMessage{{null, random, "That's already one of my fondest memories", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, random, "forget 1, Bender", false, []TestMessage{{null, random, "Ok, .*", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// Short-term memories are contextual to a user in a channel
		{davidID, general, "Bender, what is Ferris Bueller?", false, []TestMessage{{david, general, "Gosh, I have no idea .*", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, general, ";store Ferris Bueller is a Righteous Dude", false, []TestMessage{{null, general, "I'll remember .*", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, general, "Bender, what is Ferris Bueller?", false, []TestMessage{{null, general, "Ferris Bueller is a Righteous Dude", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{carolID, general, "Bender, what is Ferris Bueller?", false, []TestMessage{{carol, general, "Gosh, I have no idea .*", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, random, "Bender, what is Ferris Bueller?", false, []TestMessage{{david, random, "Gosh, I have no idea .*", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{bobID, general, "Bender, link news for nerds to https://slashdot.org", false, []TestMessage{{null, general, "Link added", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, general, ";save https://slashdot.org", false, []TestMessage{{null, general, "I already have that link", false}, {bob, general, "Do you want .*", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, general, "yes", false, []TestMessage{{null, general, "Ok, I'll replace the old one", false}, {bob, general, "What keywords or phrase .*", false}}, []Event{}, 0},
		{bobID, general, "News for Nerds, Stuff that Matters!", false, []TestMessage{{null, general, "Link added", false}}, []Event{}, 0},
		{carolID, general, "Bender, look up nerds", false, []TestMessage{{null, general, `(?s:Here's what I have .*Nerds.*)`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";link tuna casserole to https://www.allrecipes.com/recipe/17219/best-tuna-casserole/", false, []TestMessage{{null, general, `Link added`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";add it to the dinner meals list", false, []TestMessage{{alice, general, `I don't have a .*`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "yes", false, []TestMessage{{null, general, `Ok, .*`, false}}, []Event{}, 0},
		{aliceID, general, "Bender, look it up", false, []TestMessage{{null, general, `(?s:Here's what I have .*best.*)`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "add hamburgers to the list, bender", false, []TestMessage{{null, general, `Ok, I added hamburgers to the dinner meals list`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
