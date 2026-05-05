//go:build integration
// +build integration

package tbot_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
	. "github.com/lnxjedi/gopherbot/v2/bot"
	testc "github.com/lnxjedi/gopherbot/v2/connectors/test"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/groups"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/ping"
	_ "github.com/lnxjedi/gopherbot/v2/history/file"
	"github.com/lnxjedi/gopherbot/v2/integration/suites"

	// Anything referred to robot.yaml has to be compiled in
	_ "github.com/lnxjedi/gopherbot/v2/gojobs/go-bootstrap"

	_ "net/http/pprof"
)

func TestBotName(t *testing.T) {
	suite := suites.MustGet("TestBotName")
	done, conn := setup(suite.ConfigDir, "/tmp/"+suite.LogName, t)

	runRegisteredSuite(t, conn, suite)

	teardown(t, done, conn)
}

func TestBotNameHiddenCommandsUnsupportedConnector(t *testing.T) {
	done, conn, cleanup := setupWithOptions("test/membrain", "/tmp/bottest.log", testSetupOptions{
		ConnectorCapabilities: map[string]robot.ConnectorCapabilities{
			"test": {HiddenCommands: false},
		},
	}, t)

	tests := []testItem{
		{aliceID, null, "/ping", false, []TestMessage{{alice, null, "This command isn't supported with test because hidden commands are unavailable for this connector\\. Check with the robot administrator\\.", false}}, []Event{BotDirectMessage}, 0},
	}
	testcases(t, conn, tests)

	teardownWithOptions(t, done, conn, cleanup)
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
		{aliceID, general, ";hello world", false, []TestMessage{{null, general, "Hello, World!", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";HELLO   WORLD", false, []TestMessage{{null, general, "Hello, World!", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
		{aliceID, general, ";hello-world", false, []TestMessage{{null, general, "Hello, World!", false}}, []Event{CommandTaskRan, ExternalTaskRan}, 0},
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
		{aliceID, general, "help ruby, bender", false, []TestMessage{{null, general, `(?s:Command matches for keyword: ruby.*Availability: channels: random)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "ruby me, bender", false, []TestMessage{{null, general, "rubydemo/ruby not available in #general, try #random", true}}, []Event{}, 0},
		{aliceID, deadzone, "bender: echo hello world", false, []TestMessage{{null, deadzone, "echo/echo not available in #deadzone, try one of: #general, #random", true}}, []Event{}, 0},
		{aliceID, null, "hear me out", false, []TestMessage{{alice, null, "bashdemo/hear not available in direct messages, try it in any regular channel", false}}, []Event{BotDirectMessage}, 0},
		{bobID, general, ";ping", false, []TestMessage{{null, general, `(?s:I couldn't match .*More help: .*help builtin-help/help.*Try .*commands.*help <keyword>.*)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{bobID, general, ";reload", false, []TestMessage{{null, general, `(?s:I couldn't match .*Try .*commands.*help <keyword>.*)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestBuiltins(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest-builtins.log", t)

	tests := []testItem{
		{aliceID, general, ";help log", false, []TestMessage{{null, general, `(?s:Command matches for keyword: log.*Availability: direct message only)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";set log level fine", false, []TestMessage{{alice, null, "Invalid value 'fine' for 'level'; valid values: trace, debug, info, warn, error\\.", false}}, []Event{BotDirectMessage}, 0},
		{aliceID, null, ";set log lines to two", false, []TestMessage{{alice, null, "Invalid value 'two' for 'lines'; expected an integer\\.", false}}, []Event{BotDirectMessage}, 0},
		{aliceID, null, ";set log lines to 0", false, []TestMessage{{alice, null, "Lines per page of log output set to: 1", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";set log lines to 3", false, []TestMessage{{alice, null, "Lines per page of log output set to: 3", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";help info", false, []TestMessage{{null, general, `(?s:Command matches for keyword: info.*Summary: .*admins.*)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, random, ";help ruby", false, []TestMessage{{null, random, `(?s:Command matches for keyword: ruby.*Availability: channels: random)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "help", false, []TestMessage{{null, general, `(?s:Help.*Hi, I'm Bender, a staff robot\. I see you've asked for help\..*I've been programmed to perform a variety of tasks for your team.*Getting command help.*bender, help ping.*Useful discovery commands.*;help.*;commands.*;help <keyword>.*;help-all.*When I ask a follow-up question.*= uses the default value.*- cancels.*;info.*parsley@linuxjedi\.org.*)`, true}}, []Event{AmbientTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";whoami", false, []TestMessage{{null, general, "you are 'test' user 'alice/u0001', speaking in channel 'general/#general', email address: alice@example.com", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		// NOTE: Dumps are all format = Fixed, which for the test connector is ALL CAPS
		{aliceID, null, "dump robot", false, []TestMessage{{alice, null, "This command is only available as a hidden command.", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: dump robot", false, []TestMessage{{null, general, "HERE'S HOW I'VE BEEN CONFIGURED.*", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin echo", false, []TestMessage{{null, general, "ALLCHANNELS.*", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin default echo", false, []TestMessage{{null, general, "HERE'S.*", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin junk", false, []TestMessage{{null, general, "Didn't find .* junk", false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: list plugins", false, []TestMessage{{null, general, `(?s:Here are the plugins I have configured:.*builtin-admin)`, false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "/bender: list disabled plugins", false, []TestMessage{{null, general, `(?s:(Here's a list of all disabled plugins:|There are no disabled plugins))`, false}}, []Event{AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";list jobs", false, []TestMessage{{null, general, `(?s:(Here's a list of jobs for this channel:|I don't see any jobs configured for this channel))`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";list all jobs", false, []TestMessage{{null, general, `(?s:(Here's a list of all the jobs I know about:|I dont' have any jobs configured))`, false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";show log", false, []TestMessage{{alice, null, ".*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
		{aliceID, null, ";show log page 1", false, []TestMessage{{alice, null, ".*", false}}, []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestValidateUser(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest-validate-user.log", t)

	WaitForBackgroundInitsForTesting()
	GetEvents()

	conn.SendBotMessage(&testc.TestMessage{aliceID, null, "validate user bob", false, false})
	got, err := conn.GetBotMessage()
	if err != nil {
		t.Fatalf("timed out waiting for validation code reply: %v", err)
	}
	if got.User != alice || got.Channel != null {
		t.Fatalf("validation code reply target = {%s, %s}, want {%s, %s}", got.User, got.Channel, alice, null)
	}
	codeRe := regexp.MustCompile(`Validation code for 'bob': ([0-9]{7})`)
	matches := codeRe.FindStringSubmatch(got.Message)
	if len(matches) != 2 {
		t.Fatalf("validation code reply = %q", got.Message)
	}
	GetEvents()

	conn.SendBotMessage(&testc.TestMessage{bobID, null, matches[1], false, false})
	got, err = conn.GetBotMessage()
	if err != nil {
		t.Fatalf("timed out waiting for validation notification: %v", err)
	}
	if got.User != alice || got.Channel != null {
		t.Fatalf("validation notification target = {%s, %s}, want {%s, %s}", got.User, got.Channel, alice, null)
	}
	if got.Message != "User validation received: test user 'bob' has internal ID 'u0002'" {
		t.Fatalf("validation notification = %q", got.Message)
	}
	GetEvents()

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
		{aliceID, general, ";create a new grocery list", false, []TestMessage{{null, general, `(?s:I couldn't match .*create a new grocery list.*More help: /bender help lists/add.*Try /bender commands or /bender help <keyword>\.)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";add bananas to the grocery list", false, []TestMessage{{alice, general, "I don't have a 'grocery' list, do you want to create it?", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, "yes", false, []TestMessage{{null, general, "Ok, I created a new grocery list and added bananas to it", false}}, []Event{}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestHelp(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, deadzone, ";help", false, []TestMessage{{null, deadzone, `(?s:Quick help.*- /bender help <keyword> - get help for the provided <keyword>.*- /bender help <keyword> brief - compact help for a likely command.*- /bender commands - browse plugins and command groups available in this channel.*- /bender help-all - help for all commands available in this channel, including global commands.*Plugin help: /bender help <plugin>.*Exact command help: /bender help <plugin>/<command>.*Browse this channel: /bender commands)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";commands", false, []TestMessage{{null, deadzone, `(?s:Plugins and command groups available in this channel.*builtin-help.*Commands: builtin-help/commands.*Help: /bender help builtin-help or /bender help builtin-help/<command>.*Exact help: /bender help <plugin>/<command>.*Search by keyword: /bender help <plugin\|command\|keyword>)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help-all", false, []TestMessage{{null, deadzone, `(?s:Commands available in this channel \(including global\).*Command: builtin-help/help.*Usage: help <keyword>.*Exact help: /bender help builtin-help/help)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help help", false, []TestMessage{{null, deadzone, `(?s:Help for keyword: help.*Plugin help: help.*Commands:.*- help/help.*Example: ;help with robot.*More detail: /bender help help/help.*Other command matches:.*Command: builtin-help/help.*Exact help: /bender help builtin-help/help)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help knock", false, []TestMessage{{null, deadzone, `(?s:Help for keyword: knock.*Plugin help: knock.*Commands:.*- knock/knock - Starts an interactive knock-knock joke\..*Example: ;tell me a knock-knock joke.*More detail: /bender help knock/knock)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";help knock/knock", false, []TestMessage{{null, deadzone, `(?s:Command help: knock/knock.*Usage: tell me a knock-knock joke.*Availability:)`, true}}, []Event{CommandTaskRan, GoPluginRan}, 0},
		{aliceID, general, ";tell me a jok", false, []TestMessage{{null, general, `(?s:;tell me a jok looks close to knock/knock.*Try: ;tell me a knock-knock joke.*More help: /bender help knock/knock)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";knok", false, []TestMessage{{null, deadzone, `(?s:I couldn't match ;knok in channel #deadzone.*#general or #random.*/bender help knock/knock)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestHelpWithoutHiddenCommands(t *testing.T) {
	done, conn, cleanup := setupWithOptions("test/membrain", "/tmp/bottest.log", testSetupOptions{
		ConnectorCapabilities: map[string]robot.ConnectorCapabilities{
			"test": {HiddenCommands: false},
		},
	}, t)

	tests := []testItem{
		{aliceID, general, ";tell me a jok", false, []TestMessage{{null, general, `(?s:;tell me a jok looks close to knock/knock.*Try: ;tell me a knock-knock joke.*More help: ;help knock/knock)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, ";knok", false, []TestMessage{{null, deadzone, `(?s:I couldn't match ;knok in channel #deadzone.*#general or #random.*;help knock/knock)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardownWithOptions(t, done, conn, cleanup)
}

func TestHelpNoAliasWithoutHiddenCommands(t *testing.T) {
	done, conn, cleanup := setupWithOptions("test/membrain-noalias", "/tmp/bottest.log", testSetupOptions{
		ConnectorCapabilities: map[string]robot.ConnectorCapabilities{
			"test": {HiddenCommands: false},
		},
	}, t)

	tests := []testItem{
		{aliceID, general, "bender, tell me a jok", false, []TestMessage{{null, general, `(?s:bender, tell me a jok looks close to knock/knock.*Try: bender, tell me a knock-knock joke.*More help: bender, help knock/knock)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
		{aliceID, deadzone, "bender, knok", false, []TestMessage{{null, deadzone, `(?s:I couldn't match bender, knok in channel #deadzone.*#general or #random.*bender, help knock/knock)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardownWithOptions(t, done, conn, cleanup)
}

func TestHelpConnectorWithoutProtocolBotNameUsesBotInfoForHiddenHelp(t *testing.T) {
	done, conn := setup("test/membrain-nameless", "/tmp/bottest.log", t)

	tests := []testItem{
		{aliceID, general, ";tell me a jok", false, []TestMessage{{null, general, `(?s:;tell me a jok looks close to knock/knock.*Try: ;tell me a knock-knock joke.*More help: /bender help knock/knock)`, true}}, []Event{CatchAllsRan, CatchAllTaskRan, GoPluginRan}, 0},
	}
	testcases(t, conn, tests)

	teardown(t, done, conn)
}

func TestHelpGroupFiltering(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)
	defer teardown(t, done, conn)

	getHelpReply := func(user string) string {
		GetEvents()
		conn.SendBotMessage(&testc.TestMessage{User: user, Channel: general, Message: ";help lists", Threaded: false, Hidden: false})
		reply, err := conn.GetBotMessage()
		if err != nil {
			t.Fatalf("timeout waiting for help reply: %v", err)
		}
		if reply.Channel != general || !reply.Threaded {
			t.Fatalf("expected threaded help reply in #%s, got channel=%q threaded=%t", general, reply.Channel, reply.Threaded)
		}
		GetEvents()
		return reply.Message
	}

	aliceHelp := getHelpReply(aliceID)
	if !strings.Contains(aliceHelp, "lists/add") {
		t.Fatalf("expected lists help output for alice to include at least one lists command, got: %q", aliceHelp)
	}
	if strings.Contains(aliceHelp, "lists/send") {
		t.Fatalf("expected help to hide unauthorized command '[lists] send' for alice, got: %q", aliceHelp)
	}

	bobHelp := getHelpReply(bobID)
	if !strings.Contains(bobHelp, "lists/send") {
		t.Fatalf("expected help to include authorized command '[lists] send' for bob, got: %q", bobHelp)
	}
}
