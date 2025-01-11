//go:build integration
// +build integration

package bot_test

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
	testc "github.com/lnxjedi/gopherbot/v2/connectors/test"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/groups"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/ping"
	_ "github.com/lnxjedi/gopherbot/v2/history/file"

	// Anything referred to robot.yaml has to be compiled in
	_ "github.com/lnxjedi/gopherbot/v2/gojobs/go-bootstrap"

	_ "net/http/pprof"
)

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
		{aliceID, general, "reload, bender", []testc.TestMessage{{null, general, "Starting init job 'go-bootstrap'.*"}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan, ScheduledTaskRan}, 0},
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
		{aliceID, general, "ruby me, bender", []testc.TestMessage{{null, general, "No command matched in channel.*"}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{bobID, general, ";ping", []testc.TestMessage{{null, general, "No command matched in channel.*"}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{bobID, general, ";reload", []testc.TestMessage{{null, general, "No command matched in channel.*"}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBuiltins(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest-builtins.log", t)

	tests := []testItem{
		{aliceID, general, ";help log", []testc.TestMessage{{null, general, "direct message only"}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";set log lines to 0", []testc.TestMessage{{alice, null, "Lines per page of log output set to: 1"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";set log lines to 3", []testc.TestMessage{{alice, null, "Lines per page of log output set to: 3"}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";help info", []testc.TestMessage{{null, general, `;.*admins.*`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, random, ";help ruby", []testc.TestMessage{{null, random, `prove that ruby plugins work \(channels: random\)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "help", []testc.TestMessage{{null, general, "Hi,.*"}}, []Event{AmbientTaskRan, GoPluginRan}, 0},
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
		{aliceID, deadzone, ";help", []testc.TestMessage{{null, deadzone, `(?s:Command\(s\) available in this channel:\n;help <keyword> - get help for the provided <keyword>\n\n;help-all - help for all commands available in this channel, including global commands)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help-all", []testc.TestMessage{{null, deadzone, `(?s:^Command(?:[^\n]*\n){39}[^\n]*$)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help help", []testc.TestMessage{{null, deadzone, `(?s:^Command(?:[^\n]*\n){5}[^\n]*$)`}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
