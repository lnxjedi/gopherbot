//go:build integration
// +build integration

package bot_test

// memory_integration_test.go - tests that stress the robot's memory functions.

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
	testc "github.com/lnxjedi/gopherbot/v2/connectors/test"
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
		{carolID, random, ";remember slowly The Alamo", []testc.TestMessage{{null, random, "Ok, .*"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, random, ";remember Ferris Bueller", []testc.TestMessage{{null, random, "Ok, .*"}, {null, random, "committed to memory"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{bobID, random, "recall 1, Bender", []testc.TestMessage{{null, random, "Ferris Bueller"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{carolID, random, ";remember Ferris Bueller", []testc.TestMessage{{null, random, "That's already one of my fondest memories"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, random, "forget 1, Bender", []testc.TestMessage{{null, random, "Ok, .*"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// Short-term memories are contextual to a user in a channel
		{davidID, general, "Bender, what is Ferris Bueller?", []testc.TestMessage{{david, general, "Gosh, I have no idea .*"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, general, ";store Ferris Bueller is a Righteous Dude", []testc.TestMessage{{null, general, "I'll remember .*"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, general, "Bender, what is Ferris Bueller?", []testc.TestMessage{{null, general, "Ferris Bueller is a Righteous Dude"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{carolID, general, "Bender, what is Ferris Bueller?", []testc.TestMessage{{carol, general, "Gosh, I have no idea .*"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, random, "Bender, what is Ferris Bueller?", []testc.TestMessage{{david, random, "Gosh, I have no idea .*"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{bobID, general, "Bender, link news for nerds to https://slashdot.org", []testc.TestMessage{{null, general, "Link added"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, general, ";save https://slashdot.org", []testc.TestMessage{{null, general, "I already have that link"}, {bob, general, "Do you want .*"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, general, "yes", []testc.TestMessage{{null, general, "Ok, I'll replace the old one"}, {bob, general, "What keywords or phrase .*"}}, []Event{}, 0},
		{bobID, general, "News for Nerds, Stuff that Matters!", []testc.TestMessage{{null, general, "Link added"}}, []Event{}, 0},
		{carolID, general, "Bender, look up nerds", []testc.TestMessage{{null, general, `(?s:Here's what I have .*Nerds.*)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";link tuna casserole to https://www.allrecipes.com/recipe/17219/best-tuna-casserole/", []testc.TestMessage{{null, general, `Link added`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";add it to the dinner meals list", []testc.TestMessage{{alice, general, `I don't have a .*`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "yes", []testc.TestMessage{{null, general, `Ok, .*`}}, []Event{}, 0},
		{aliceID, general, "Bender, look it up", []testc.TestMessage{{null, general, `(?s:Here's what I have .*best.*)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "add hamburgers to the list, bender", []testc.TestMessage{{null, general, `Ok, I added hamburgers to the dinner meals list`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
