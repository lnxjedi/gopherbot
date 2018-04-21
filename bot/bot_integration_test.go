// +build integration

package bot_test

/*
bot_integration_test.go - setup and initialization of "black box" integration testing.

Run integration tests with:
$ go test -v --tags 'test integration' -cover -race -coverprofile coverage.out -coverpkg ./... ./bot

Run specific tests with e.g.:
$ go test -run MessageMatch -v --tags 'test integration' -cover -race -coverprofile coverage.out -coverpkg ./... ./bot

Generate coverage statistics report with:
$ go tool cover -html=coverage.out -o coverage.html

(eventual) Setup for "clear box" testing of bot internals is in bot_test.go
*/

import (
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/lnxjedi/gopherbot/bot"
	_ "github.com/lnxjedi/gopherbot/brains/file"
	_ "github.com/lnxjedi/gopherbot/brains/mem"
	testc "github.com/lnxjedi/gopherbot/connectors/test"
	_ "github.com/lnxjedi/gopherbot/goplugins/groups"
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/links"
	_ "github.com/lnxjedi/gopherbot/goplugins/lists"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"

	// Enable profiling. You can shrink the binary by removing this, but if the
	// robot ever stops responding for any reason, it's handy for getting a
	// dump of all goroutines.
	_ "net/http/pprof"
)

type testItem struct {
	user, channel, message string
	replies                []testc.TestMessage // note: TestMessage.Message -> regex
	events                 []Event
	pause                  int // time in milliseconds to pause after test item
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
		if test.pause > 0 {
			time.Sleep(time.Millisecond * time.Duration(test.pause))
		}
	}
}

func TestBotName(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, null, "ping, bender", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, ";ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "bender ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping, bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, ";ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "bender, ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "@bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping @bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping;", []testc.TestMessage{}, []Event{}, 0},
		{bob, general, "bender: echo hello world", []testc.TestMessage{{null, general, "hello world"}}, []Event{CommandPluginRan, ScriptPluginRan}, 0},
		// When you forget to address the robot, you can say it's name
		{alice, general, "ping", []testc.TestMessage{}, []Event{}, 200},
		{alice, general, "bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping", []testc.TestMessage{}, []Event{}, 200},
		{alice, general, ";", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBotNoName(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, null, ";ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, general, ";ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping;", []testc.TestMessage{}, []Event{}, 0},
		{bob, general, "bender: echo hello world", []testc.TestMessage{{null, general, "hello world"}}, []Event{CommandPluginRan, ScriptPluginRan}, 0},
		// When you forget to address the robot, you can say it's name
		{alice, general, "ping", []testc.TestMessage{}, []Event{}, 200},
		{alice, general, ";", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping", []testc.TestMessage{}, []Event{}, 100},
		{alice, general, "hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientPluginRan, ScriptPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBotNoAlias(t *testing.T) {
	done, conn := setup("cfg/test/membrain-noalias", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, null, "ping, bender", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "bender ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping, bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "bender, ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "@bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping @bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{bob, general, "bender: echo hello world", []testc.TestMessage{{null, general, "hello world"}}, []Event{CommandPluginRan, ScriptPluginRan}, 0},
		// When you forget to address the robot, you can say it's name
		{alice, general, "ping", []testc.TestMessage{}, []Event{}, 200},
		{alice, general, "bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ping", []testc.TestMessage{}, []Event{}, 100},
		{alice, general, "hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientPluginRan, ScriptPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestReload(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, general, "reload, bender", []testc.TestMessage{{alice, general, "Configuration reloaded successfully"}}, []Event{CommandPluginRan, GoPluginRan, AdminCheckPassed}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestMessageMatch(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, general, "hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientPluginRan, ScriptPluginRan}, 0},
		{alice, general, ";hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientPluginRan, ScriptPluginRan}, 0},
		{alice, null, "hello robot", []testc.TestMessage{{alice, null, "Hello, World!"}}, []Event{BotDirectMessage, AmbientPluginRan, ScriptPluginRan}, 0},
		{alice, null, "bender, hello robot", []testc.TestMessage{{alice, null, "Hello, World!"}}, []Event{BotDirectMessage, AmbientPluginRan, ScriptPluginRan}, 0},
		{alice, general, "ping", []testc.TestMessage{}, []Event{}, 100},
		{alice, general, ";hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientPluginRan, ScriptPluginRan}, 100},
		{alice, general, "bender", []testc.TestMessage{{null, general, `Yes\?`}}, []Event{}, 0},
		{alice, random, "hello robot", []testc.TestMessage{{null, random, "Hello, World!"}}, []Event{AmbientPluginRan, ScriptPluginRan}, 100},
		{alice, random, ";hello robot", []testc.TestMessage{{null, random, "I'm here"}}, []Event{CommandPluginRan, ScriptPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestVisibility(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, general, "help ruby, bender", []testc.TestMessage{{null, general, `bender, ruby .*random\)`}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "ruby me, bender", []testc.TestMessage{{alice, general, "Sorry, that didn't match.*"}}, []Event{CatchAllsRan, GoPluginRan}, 0},
		{bob, general, ";ping", []testc.TestMessage{{bob, general, "Sorry, that didn't match.*"}}, []Event{CatchAllsRan, GoPluginRan}, 0},
		{bob, general, ";reload", []testc.TestMessage{{bob, general, "Sorry, that didn't match.*"}}, []Event{CatchAllsRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBuiltins(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, general, ";help log", []testc.TestMessage{{null, general, "direct message only"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, null, ";set log lines to 3", []testc.TestMessage{{alice, null, "Lines per page of log output set to: 3"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, ";set log lines to 0", []testc.TestMessage{{alice, null, "Lines per page of log output set to: 1"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, ";show log", []testc.TestMessage{{alice, null, ".*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, ";show log page 1", []testc.TestMessage{{alice, null, ".*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, general, ";help info", []testc.TestMessage{{null, general, "bender,.*admins"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, random, ";help ruby", []testc.TestMessage{{null, random, `(?m:Command.*\n.*random\))`}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, ";help", []testc.TestMessage{{alice, general, `\(the help.*private message\)`}, {alice, null, "bender,.*"}}, []Event{CommandPluginRan, GoPluginRan}, 0},
		{alice, general, "help", []testc.TestMessage{{alice, general, "I've sent.*myself"}, {alice, null, "Hi,.*"}}, []Event{AmbientPluginRan, GoPluginRan}, 0},
		{alice, null, "dump robot", []testc.TestMessage{{alice, null, "Here's how I've been configured.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "dump plugin echo", []testc.TestMessage{{alice, null, "AllChannels.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "dump plugin default echo", []testc.TestMessage{{alice, null, "Here's.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "dump plugin rubydemo", []testc.TestMessage{{alice, null, "AllChannels.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "dump plugin default rubydemo", []testc.TestMessage{{alice, null, "Here's.*"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
		{alice, null, "dump plugin junk", []testc.TestMessage{{alice, null, "Didn't find .* junk"}}, []Event{BotDirectMessage, CommandPluginRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestPrompting(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{carol, general, "Bender, listen to me", []testc.TestMessage{{carol, null, "Ok, .*"}}, []Event{CommandPluginRan, ScriptPluginRan}, 0},
		{carol, null, "You're pretty cool", []testc.TestMessage{{carol, null, "I hear .*cool\""}}, []Event{BotDirectMessage}, 0},
		{bob, general, "hear me out, Bender", []testc.TestMessage{{bob, general, "Well ok then.*"}}, []Event{CommandPluginRan, ScriptPluginRan}, 0},
		{bob, general, "I like kittens", []testc.TestMessage{{bob, general, "Ok, I hear you saying \"I like kittens\".*"}}, []Event{}, 0},
		// wait ask waits a second before prompting; in 2 seconds it'll message the test to answer the second question first
		{david, general, ";waitask", []testc.TestMessage{}, []Event{}, 200},
		// ask now asks a question right away, but we don't reply until the command above tells us to - by which time the first command has prompted, but now has to wait
		{david, general, ";asknow", []testc.TestMessage{{david, general, `Do you like puppies\?`}, {null, general, `ok - answer puppies`}}, []Event{CommandPluginRan, ScriptPluginRan, CommandPluginRan, ScriptPluginRan}, 0},
		{david, general, "yes", []testc.TestMessage{{david, general, `Do you like kittens\?`}, {null, general, `I like puppies too!`}}, []Event{}, 0},
		{david, general, "yes", []testc.TestMessage{{null, general, `I like kittens too!`}}, []Event{}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

// pythondemo is active in general, rubydemo in random; pythondemo is trusted by
// echo.sh, rubydemo is not.
func TestCalling(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{alice, general, ";bashecho foo bar baz", []testc.TestMessage{{null, general, "foo bar baz"}}, []Event{CommandPluginRan, ScriptPluginRan}, 0},
		{alice, random, ";bashecho foo bar baz", []testc.TestMessage{{null, random, "Sorry, .*"}}, []Event{CommandPluginRan, ScriptPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
