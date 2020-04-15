// +build integration

package bot_test

/*
bot_integration_test.go - setup and initialization of "black box" integration testing.

Run integration tests with:
$ go test -v --tags 'test integration' ./test

Run specific tests with e.g.:
$ go test -run MessageMatch -v --tags 'test integration' ./test

To run tests with static builds and modules from vendor:
$ CGO_ENABLED=0 go test -v --tags 'test integration netgo osusergo static_build' -mod vendor ./test

Generate coverage statistics report with:
$ go tool cover -html=coverage.out -o coverage.html

Check status of goroutines if tests get hung up
$ go tool pprof http://localhost:8889/debug/pprof/goroutine
...
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) list lnxjedi
Total: 11
ROUTINE ======================== github.com/lnxjedi/gopherbot/bot...

(eventual) Setup for "clear box" testing of bot internals is in bot_test.go
*/

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/lnxjedi/gopherbot/bot"
	testc "github.com/lnxjedi/gopherbot/connectors/test"
	_ "github.com/lnxjedi/gopherbot/goplugins/groups"
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/links"
	_ "github.com/lnxjedi/gopherbot/goplugins/lists"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"
	_ "github.com/lnxjedi/gopherbot/history/file"

	_ "net/http/pprof"
)

var testInstallPath string

// Environment setting(s) for expanding installed conf/robot.yaml
func init() {
	os.Setenv("GOPHER_PROTOCOL", "test")
	wd, _ := os.Getwd()
	testInstallPath = filepath.Dir(wd)
}

type testItem struct {
	user, channel, message string
	replies                []testc.TestMessage // note: TestMessage.Message -> regex
	events                 []Event
	pause                  int // time in milliseconds to pause after test item
}

// NOTE: integration tests are closely tied to the configuration in test/...

// Cast of Users
const alice = "alice"
const bob = "bob"
const carol = "carol"
const david = "david"
const erin = "erin"
const aliceID = "u0001"
const bobID = "u0002"
const carolID = "u0003"
const davidID = "u0004"
const erinID = "u0005"

// When the robot doesn't address the user specifically, or sends a DM
const null = ""

// ... and the Channels they play in
const general = "general"
const random = "random"
const bottest = "bottest"
const deadzone = "deadzone"

func setup(cfgdir, logfile string, t *testing.T) (<-chan bool, *testc.TestConnector) {
	os.Setenv("GOPHER_ENCRYPTION_KEY", "gopherbot-integration-tests-brain-key")
	testVer := VersionInfo{"test", "(unknown)"}

	testc.ExportTest.Lock()
	testc.ExportTest.Test = t
	testc.ExportTest.Unlock()

	done, tconn := StartTest(testVer, cfgdir, logfile, t)
	testConnector := tconn.(*testc.TestConnector)

	return done, testConnector
}

func teardown(t *testing.T, done <-chan bool, conn *testc.TestConnector) {
	// Alice is a bot admin who can order the bot to quit in #general
	conn.SendBotMessage(&testc.TestMessage{aliceID, null, "quit"})

	// Now we wait for the connection to finish
	<-done
	ws := filepath.Join(testInstallPath, "test", "workspace")
	if err := os.RemoveAll(ws); err != nil {
		fmt.Printf("Removing temporary workspace: %v\n", err)
	}

	evOk := true
	ev := GetEvents()
	want := []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}
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
		// Clear out start-up events
		GetEvents()
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
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, null, "ping, bender", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "bender ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping, bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		// This was matching too often when a user was talking about (instead of to) the robot
		//{aliceID, general, "ping bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "bender, ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "@bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping, @bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping;", []testc.TestMessage{}, []Event{}, 0},
		{bobID, general, "bender: echo hello world", []testc.TestMessage{{null, general, "Sure thing: hello world"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// When you forget to address the robot, you can say it's name
		{aliceID, general, "ping", []testc.TestMessage{}, []Event{}, 300},
		{aliceID, general, "bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping", []testc.TestMessage{}, []Event{}, 300},
		{aliceID, general, ";", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBotNoName(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, null, ";ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping;", []testc.TestMessage{}, []Event{}, 0},
		{bobID, general, "bender: echo hello world", []testc.TestMessage{{null, general, "hello world"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// When you forget to address the robot, you can say it's name
		{aliceID, general, "ping", []testc.TestMessage{}, []Event{}, 500},
		{aliceID, general, ";", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping", []testc.TestMessage{}, []Event{}, 100},
		{aliceID, general, "hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBotNoAlias(t *testing.T) {
	done, conn := setup("test/membrain-noalias", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, null, "ping, bender", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "bender ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "ping", []testc.TestMessage{{alice, null, "PONG"}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping, bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		// Support for bare names at end removed
		//{aliceID, general, "ping bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "bender, ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "@bender ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping, @bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, general, "bender: echo hello world", []testc.TestMessage{{null, general, "hello world"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// When you forget to address the robot, you can say it's name
		{aliceID, general, "ping", []testc.TestMessage{}, []Event{}, 200},
		{aliceID, general, "bender", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping", []testc.TestMessage{}, []Event{}, 100},
		{aliceID, general, "hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestReload(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, "reload, bender", []testc.TestMessage{{alice, general, "Configuration reloaded successfully"}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestMessageMatch(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, "hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientTaskRan, ExternalTaskRan}, 0},
		{aliceID, null, "hello robot", []testc.TestMessage{{alice, null, "Hello, World!"}}, []Event{BotDirectMessage, AmbientTaskRan, ExternalTaskRan}, 0},
		{aliceID, null, "bender, hello robot", []testc.TestMessage{{alice, null, "Hello, World!"}}, []Event{BotDirectMessage, AmbientTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, "ping", []testc.TestMessage{}, []Event{}, 100},
		{aliceID, general, ";hello robot", []testc.TestMessage{{null, general, "Hello, World!"}}, []Event{AmbientTaskRan, ExternalTaskRan}, 100},
		{aliceID, general, "bender", []testc.TestMessage{{null, general, `Yes\?`}}, []Event{}, 0},
		{aliceID, random, "hello robot", []testc.TestMessage{{null, random, "Hello, World!"}}, []Event{AmbientTaskRan, ExternalTaskRan}, 100},
		{aliceID, random, ";hello robot", []testc.TestMessage{{null, random, "I'm here"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestVisibility(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, "help ruby, bender", []testc.TestMessage{{null, general, `bender, ruby .*random\)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ruby me, bender", []testc.TestMessage{{alice, general, "Sorry, that didn't match.*"}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{bobID, general, ";ping", []testc.TestMessage{{bob, general, "Sorry, that didn't match.*"}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{bobID, general, ";reload", []testc.TestMessage{{bob, general, "Sorry, that didn't match.*"}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBuiltins(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, ";help log", []testc.TestMessage{{null, general, "direct message only"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";set log lines to 3", []testc.TestMessage{{alice, null, "Lines per page of log output set to: 3"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";set log lines to 0", []testc.TestMessage{{alice, null, "Lines per page of log output set to: 1"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";help info", []testc.TestMessage{{null, general, "bender,.*admins"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, random, ";help ruby", []testc.TestMessage{{null, random, `(?m:Command.*\n.*random\))`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";help", []testc.TestMessage{{alice, general, `\(the help.*private message\)`}, {alice, null, "bender,.*"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "help", []testc.TestMessage{{alice, general, "I've sent.*myself"}, {alice, null, "Hi,.*"}}, []Event{AmbientTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";whoami", []testc.TestMessage{{null, general, "you are 'test' user 'alice/u0001', speaking in channel 'general/#general', email address: alice@example.com"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		// NOTE: Dumps are all format = Fixed, which for the test connector is ALL CAPS
		{aliceID, null, "dump robot", []testc.TestMessage{{alice, null, "HERE'S HOW I'VE BEEN CONFIGURED.*"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin echo", []testc.TestMessage{{alice, null, "ALLCHANNELS.*"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin default echo", []testc.TestMessage{{alice, null, "HERE'S.*"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin rubydemo", []testc.TestMessage{{alice, null, "ALLCHANNELS.*"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin default rubydemo", []testc.TestMessage{{alice, null, "HERE'S.*"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin junk", []testc.TestMessage{{alice, null, "Didn't find .* junk"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";show log", []testc.TestMessage{{alice, null, ".*"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";show log page 1", []testc.TestMessage{{alice, null, ".*"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestPrompting(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{carolID, general, "Bender, listen to me", []testc.TestMessage{{carol, null, "Ok, .*"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{carolID, null, "You're pretty cool", []testc.TestMessage{{carol, null, "I hear .*cool\""}}, []Event{BotDirectMessage}, 0},
		{bobID, general, "hear me out, Bender", []testc.TestMessage{{bob, general, "Well ok then.*"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{bobID, general, "I like kittens", []testc.TestMessage{{bob, general, "Ok, I hear you saying \"I like kittens\".*"}}, []Event{}, 0},
		// wait ask waits a second before prompting; in 2 seconds it'll message the test to answer the second question first
		{davidID, general, ";waitask", []testc.TestMessage{}, []Event{}, 200},
		// ask now asks a question right away, but we don't reply until the command above tells us to - by which time the first command has prompted, but now has to wait
		{davidID, general, ";asknow", []testc.TestMessage{{david, general, `Do you like puppies\?`}, {null, general, `ok - answer puppies`}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, general, "yes", []testc.TestMessage{{david, general, `Do you like kittens\?`}, {null, general, `I like puppies too!`}}, []Event{}, 0},
		{davidID, general, "yes", []testc.TestMessage{{null, general, `I like kittens too!`}}, []Event{}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestFormatting(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, ";format fixed", []testc.TestMessage{{null, general, "_ITALICS_ <ONE> \\*BOLD\\* `CODE` @PARSLEY"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";format variable", []testc.TestMessage{{null, general, "_italics_ <one> \\*bold\\* `code` @parsley"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";format raw", []testc.TestMessage{{null, general, "_Italics_ <One> \\*Bold\\* `Code` @parsley"}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestHelp(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		// Took a while to get the regex right; should be # of help msgs * 2 - 1; e.g. 10 lines -> 19
		// NOTE: the default 'help' output is now too long for in-channel reply
		{aliceID, deadzone, ";help", []testc.TestMessage{{alice, deadzone, `\(the help output was pretty long, so I sent you a private message\)`}, {alice, null, `(?s:^Command(?:[^\n]*\n){31}[^\n]*$)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help help", []testc.TestMessage{{null, deadzone, `(?s:^Command(?:[^\n]*\n){3}[^\n]*$)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
