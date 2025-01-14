//go:build integration
// +build integration

package tbot_test

import (
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
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
		{aliceID, null, "ping, bender", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";ping", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "bender ping", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "ping", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping, bender", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";ping", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "bender ping", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		// This was matching too often when a user was talking about (instead of to) the robot
		//{aliceID, general, "ping bender", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "bender, ping", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "@bender ping", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping, @bender", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping;", false, []TestMessage{}, []Event{}, 0},
		{bobID, general, "bender: echo hello world", false, []TestMessage{{null, general, "Sure thing: hello world", true}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// Hidden echo command
		{bobID, general, "/bender: echo hello world", false, []TestMessage{{null, general, "(Sure thing: hello world)", true}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// When you forget to address the robot, you can say it's name
		{aliceID, general, "ping", false, []TestMessage{}, []Event{}, 300},
		{aliceID, general, "bender", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping", false, []TestMessage{}, []Event{}, 300},
		{aliceID, general, ";", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBotNoName(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, null, ";ping", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "ping", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";ping", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping;", false, []TestMessage{}, []Event{}, 0},
		{bobID, general, "bender: echo hello world", false, []TestMessage{{null, general, "hello world", true}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// When you forget to address the robot, you can say it's name
		{aliceID, general, "ping", false, []TestMessage{}, []Event{}, 500},
		{aliceID, general, ";", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping", false, []TestMessage{}, []Event{}, 100},
		{aliceID, general, "hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []Event{AmbientTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBotNoAlias(t *testing.T) {
	done, conn := setup("test/membrain-noalias", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, null, "ping, bender", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "bender ping", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "ping", false, []TestMessage{{alice, null, "PONG", false}}, []Event{BotDirectMessage, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping, bender", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "bender ping", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		// Support for bare names at end removed
		//{aliceID, general, "ping bender", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "bender, ping", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "@bender ping", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping, @bender", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{bobID, general, "bender: echo hello world", false, []TestMessage{{null, general, "hello world", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		// When you forget to address the robot, you can say it's name
		{aliceID, general, "ping", false, []TestMessage{}, []Event{}, 200},
		{aliceID, general, "bender", false, []TestMessage{{alice, general, "PONG", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ping", false, []TestMessage{}, []Event{}, 100},
		{aliceID, general, "hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []Event{AmbientTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestReload(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, "reload, bender", false, []TestMessage{{null, general, "Starting init job 'go-bootstrap'.*", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan, ScheduledTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestMessageMatch(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, "hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []Event{AmbientTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []Event{AmbientTaskRan, ExternalTaskRan}, 0},
		{aliceID, null, "hello robot", false, []TestMessage{{alice, null, "Hello, World!", false}}, []Event{BotDirectMessage, AmbientTaskRan, ExternalTaskRan}, 0},
		{aliceID, null, "bender, hello robot", false, []TestMessage{{alice, null, "Hello, World!", false}}, []Event{BotDirectMessage, AmbientTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, "ping", false, []TestMessage{}, []Event{}, 100},
		{aliceID, general, ";hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []Event{AmbientTaskRan, ExternalTaskRan}, 100},
		{aliceID, general, "bender", false, []TestMessage{{null, general, `Yes\?`, false}}, []Event{}, 0},
		{aliceID, random, "hello robot", false, []TestMessage{{null, random, "Hello, World!", false}}, []Event{AmbientTaskRan, ExternalTaskRan}, 100},
		{aliceID, random, ";hello robot", false, []TestMessage{{null, random, "I'm here", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestVisibility(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, "help ruby, bender", false, []TestMessage{{null, general, `bender, ruby .*random\)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ruby me, bender", false, []TestMessage{{null, general, "No command matched in channel.*", true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{bobID, general, ";ping", false, []TestMessage{{null, general, "No command matched in channel.*", true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{bobID, general, ";reload", false, []TestMessage{{null, general, "No command matched in channel.*", true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBuiltins(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest-builtins.log", t)

	tests := []testItem{
		{aliceID, general, ";help log", false, []TestMessage{{null, general, "direct message only", true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";set log lines to 0", false, []TestMessage{{alice, null, "Lines per page of log output set to: 1", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";set log lines to 3", false, []TestMessage{{alice, null, "Lines per page of log output set to: 3", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";help info", false, []TestMessage{{null, general, `;.*admins.*`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, random, ";help ruby", false, []TestMessage{{null, random, `prove that ruby plugins work \(channels: random\)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "help", false, []TestMessage{{null, general, "Hi,.*", true}}, []Event{AmbientTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";whoami", false, []TestMessage{{null, general, "you are 'test' user 'alice/u0001', speaking in channel 'general/#general', email address: alice@example.com", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		// NOTE: Dumps are all format = Fixed, which for the test connector is ALL CAPS
		{aliceID, null, "dump robot", false, []TestMessage{{alice, null, "HERE'S HOW I'VE BEEN CONFIGURED.*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin echo", false, []TestMessage{{alice, null, "ALLCHANNELS.*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin default echo", false, []TestMessage{{alice, null, "HERE'S.*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin rubydemo", false, []TestMessage{{alice, null, "ALLCHANNELS.*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin default rubydemo", false, []TestMessage{{alice, null, "HERE'S.*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, "dump plugin junk", false, []TestMessage{{alice, null, "Didn't find .* junk", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";show log", false, []TestMessage{{alice, null, ".*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";show log page 1", false, []TestMessage{{alice, null, ".*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestPrompting(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{carolID, general, "Bender, listen to me", false, []TestMessage{{carol, null, "Ok, .*", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{carolID, null, "You're pretty cool", false, []TestMessage{{carol, null, "I hear .*cool\"", false}}, []Event{BotDirectMessage}, 0},
		{bobID, general, "hear me out, Bender", false, []TestMessage{{bob, general, "Well ok then.*", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{bobID, general, "I like kittens", false, []TestMessage{{bob, general, "Ok, I hear you saying \"I like kittens\".*", false}}, []Event{}, 0},
		// wait ask waits a second before prompting; in 2 seconds it'll message the test to answer the second question first
		{davidID, general, ";waitask", false, []TestMessage{}, []Event{}, 200},
		// ask now asks a question right away, but we don't reply until the command above tells us to - by which time the first command has prompted, but now has to wait
		{davidID, general, ";asknow", false, []TestMessage{{david, general, `Do you like puppies\?`, false}, {null, general, `ok - answer puppies`, false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{davidID, general, "yes", false, []TestMessage{{david, general, `Do you like kittens\?`, false}, {null, general, `I like puppies too!`, false}}, []Event{}, 0},
		{davidID, general, "yes", false, []TestMessage{{null, general, `I like kittens too!`, false}}, []Event{}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestFormatting(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, ";format fixed", false, []TestMessage{{null, general, "_ITALICS_ <ONE> \\*BOLD\\* `CODE` @PARSLEY", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";format variable", false, []TestMessage{{null, general, "_italics_ <one> \\*bold\\* `code` @parsley", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";format raw", false, []TestMessage{{null, general, "_Italics_ <One> \\*Bold\\* `Code` @parsley", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestDevel(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, ";add bananas to the grocery list", false, []TestMessage{{alice, general, "I don't have a 'grocery' list, do you want to create it?", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "yes", false, []TestMessage{{null, general, "Ok, I created a new grocery list and added bananas to it", false}}, []Event{}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestHelp(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		// Took a while to get the regex right; should be # of help msgs * 2 - 1; e.g. 10 lines -> 19
		// NOTE: the default 'help' output is now too long for in-channel reply
		{aliceID, deadzone, ";help", false, []TestMessage{{null, deadzone, `(?s:Command\(s\) available in this channel:\n;help <keyword> - get help for the provided <keyword>\n\n;help-all - help for all commands available in this channel, including global commands)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help-all", false, []TestMessage{{null, deadzone, `(?s:^Command(?:[^\n]*\n){39}[^\n]*$)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help help", false, []TestMessage{{null, deadzone, `(?s:^Command(?:[^\n]*\n){5}[^\n]*$)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}
