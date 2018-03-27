// +build integration

package bot_test

/*
bot_integration_test.go - setup and initialization of "black box" integration testing.

Setup for "clear box" testing of bot internals is in bot_test.go
*/

import (
	"regexp"
	"strings"
	"testing"

	. "github.com/lnxjedi/gopherbot/bot"
	_ "github.com/lnxjedi/gopherbot/brains/file"
	_ "github.com/lnxjedi/gopherbot/brains/mem"
	testc "github.com/lnxjedi/gopherbot/connectors/test"
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/links"
	_ "github.com/lnxjedi/gopherbot/goplugins/lists"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"
)

type testItem struct {
	user, channel, message string
	replies                []testc.TestMessage // note: TestMessage.Message -> regex
	events                 []Event
}

// NOTE: integration tests are closely tied to the configuration in cfg/test/...

// Cast of Users
const alice = "alice"
const bob = "bob"
const carol = "carol"
const david = "david"
const erin = "erin"

// When the robot doesn't address the user specifically, or sends a DM
const null = ""

// ... and the Channels they play in
const general = "general"
const random = "random"
const bottest = "bottest"

func setup(cfgdir, logfile string, t *testing.T) (<-chan struct{}, *testc.TestConnector) {
	done, tconn := StartTest(cfgdir, logfile, t)
	testConnector := tconn.(*testc.TestConnector)
	testConnector.SetTest(t)

	return done, testConnector
}

func teardown(t *testing.T, done <-chan struct{}, conn *testc.TestConnector) {
	// Alice is a bot admin who can order the bot to quit in #general
	conn.SendBotMessage(&testc.TestMessage{alice, general, ";quit"})

	// Now we wait for the connection to finish
	<-done

	evOk := true
	ev := GetEvents()
	want := []Event{CommandPluginRan, GoPluginRan, AdminCheckPassed}
	if len(*ev) != len(want) {
		evOk = false
	} else {
		for i, e := range *ev {
			if e != want[i] {
				evOk = false
			}
		}
	}
	if !evOk {
		gevs := make([]string, len(*ev))
		for i, e := range *ev {
			gevs[i] = e.String()
		}
		wevs := make([]string, len(want))
		for i, e := range want {
			wevs[i] = e.String()
		}
		t.Errorf("FAILED teardown events; want: \"%s\"; got: %s\n", strings.Join(wevs, ", "), strings.Join(gevs, ", "))
	}
}

func testcases(t *testing.T, conn *testc.TestConnector, tests []testItem) {
	for _, test := range tests {
		conn.SendBotMessage(&testc.TestMessage{test.user, test.channel, test.message})
		for _, want := range test.replies {
			if re, err := regexp.Compile(want.Message); err != nil {
				t.Errorf("FAILED: regex \"%s\" didn't compile: %v", want.Message, err)
			} else {
				got, err := conn.GetBotMessage()
				if err != nil {
					t.Errorf("FAILED timeout waiting for reply from robot; want: \"%s\"", want.Message)
				} else {
					if !re.MatchString(got.Message) {
						t.Errorf("FAILED message regex match; want: \"%s\", got: \"%s\"", want.Message, got.Message)
					} else {
						if got.User != want.User || got.Channel != want.Channel {
							t.Errorf("FAILED user/channel match; want u:%s, c:%s; got u:%s,c:%s", want.User, want.Channel, got.User, got.Channel)
						}
					}
				}
			}
		}
		ev := GetEvents()
		evOk := true
		if len(*ev) != len(test.events) {
			evOk = false
		} else {
			for i, e := range *ev {
				if e != test.events[i] {
					evOk = false
				}
			}
		}
		if !evOk {
			wevs := make([]string, len(test.events))
			for i, e := range test.events {
				wevs[i] = e.String()
			}
			gevs := make([]string, len(*ev))
			for i, e := range *ev {
				gevs[i] = e.String()
			}
			t.Errorf("FAILED emitted events; want: \"%s\"; got: %s\n", strings.Join(wevs, ", "), strings.Join(gevs, ", "))
		}
	}
}

func TestBotName(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "test.log", t)

	tests := []testItem{
		{alice, general, "ping, bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, ";ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, "bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, "ping bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, "bender, ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, "@bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, "ping @bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, "ping;", []testc.TestMessage{}, []Event{}},
		// When you forget to address the robot, you can say it's name
		{alice, general, "ping", []testc.TestMessage{}, []Event{}},
		{alice, general, "bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestReload(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "test.log", t)

	tests := []testItem{
		{alice, general, "reload, bender", []testc.TestMessage{{alice, general, "Configuration reloaded successfully"}}, []Event{CommandPluginRan, GoPluginRan, AdminCheckPassed}},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestPrompting(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "test.log", t)

	tests := []testItem{
		{carol, general, "Bender, listen to me", []testc.TestMessage{{carol, null, "Ok, .*"}}, []Event{CommandPluginRan, ScriptPluginRan}},
		{carol, null, "You're pretty cool", []testc.TestMessage{{carol, null, "I hear .*cool\""}}, []Event{BotDirectMessage}},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBuiltins(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "test.log", t)

	tests := []testItem{
		{alice, general, ";help info", []testc.TestMessage{{null, general, "bender,.*admins"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, random, ";help ruby", []testc.TestMessage{{null, random, `(?m:Command.*\n.*random\))`}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, ";help", []testc.TestMessage{{alice, general, `\(the help.*private message\)`}, {alice, null, "bender,.*"}}, []Event{CommandPluginRan, GoPluginRan}},
		{alice, general, "help", []testc.TestMessage{{alice, general, "I've sent.*myself"}, {alice, null, "Hi,.*"}}, []Event{AmbientPluginRan, GoPluginRan}},
		{alice, null, "dump robot", []testc.TestMessage{{alice, null, "Here's how I've been configured.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}},
		{alice, null, "dump plugin echo", []testc.TestMessage{{alice, null, "AllChannels.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}},
		{alice, null, "dump plugin default echo", []testc.TestMessage{{alice, null, "Here's.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}},
		{alice, null, "dump plugin rubydemo", []testc.TestMessage{{alice, null, "AllChannels.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}},
		{alice, null, "dump plugin default rubydemo", []testc.TestMessage{{alice, null, "Here's.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}},
		{alice, null, "dump plugin junk", []testc.TestMessage{{alice, null, "Didn't find .* junk"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestMemory(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "test.log", t)

	tests := []testItem{
		{carol, random, ";remember slowly The Alamo", []testc.TestMessage{{null, random, "Ok, .*"}}, []Event{CommandPluginRan, ScriptPluginRan}},
		{alice, random, ";remember Ferris Bueller", []testc.TestMessage{{null, random, "Ok, .*"}, {null, random, "committed to memory"}}, []Event{CommandPluginRan, ScriptPluginRan}},
		{bob, random, "recall 1, Bender", []testc.TestMessage{{null, random, "Ferris Bueller"}}, []Event{CommandPluginRan, ScriptPluginRan}},
		{david, random, "forget 1, Bender", []testc.TestMessage{{null, random, "Ok, .*"}}, []Event{CommandPluginRan, ScriptPluginRan}},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
