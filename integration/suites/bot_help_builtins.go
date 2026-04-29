package suites

import (
	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/bot"
)

func init() {
	Register(Suite{
		Name:      "TestBuiltins",
		ConfigDir: "test/membrain",
		LogName:   "bottest-builtins.log",
		Cases: legacyCases([]testItem{
			{aliceID, general, ";help log", false, []TestMessage{{null, general, `(?s:Command matches for keyword: log.*Availability: direct message only)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, null, ";set log lines to 0", false, []TestMessage{{alice, null, "Lines per page of log output set to: 1", false}}, []bot.Event{bot.BotDirectMessage, bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, null, ";set log lines to 3", false, []TestMessage{{alice, null, "Lines per page of log output set to: 3", false}}, []bot.Event{bot.BotDirectMessage, bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, ";help info", false, []TestMessage{{null, general, `(?s:Command matches for keyword: info.*Summary: .*admins.*)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, random, ";help ruby", false, []TestMessage{{null, random, `(?s:Command matches for keyword: ruby.*Availability: channels: random)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "help", false, []TestMessage{{null, general, `(?s:Help.*Hi, I'm Bender, a staff robot\. I see you've asked for help\..*I've been programmed to perform a variety of tasks for your team.*Getting command help.*bender, help ping.*Useful discovery commands.*;help.*;commands.*;help <keyword>.*;help-all.*When I ask a follow-up question.*= uses the default value.*- cancels.*;info.*parsley@linuxjedi\.org.*)`, true}}, []bot.Event{bot.AmbientTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, ";whoami", false, []TestMessage{{null, general, "you are 'test' user 'alice/u0001', speaking in channel 'general/#general', email address: alice@example.com", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, null, "dump robot", false, []TestMessage{{alice, null, "This command is only available as a hidden command.", false}}, []bot.Event{bot.BotDirectMessage, bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "/bender: dump robot", false, []TestMessage{{null, general, "HERE'S HOW I'VE BEEN CONFIGURED.*", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "/bender: dump plugin echo", false, []TestMessage{{null, general, "ALLCHANNELS.*", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "/bender: dump plugin default echo", false, []TestMessage{{null, general, "HERE'S.*", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "/bender: dump plugin junk", false, []TestMessage{{null, general, "Didn't find .* junk", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "/bender: list plugins", false, []TestMessage{{null, general, `(?s:Here are the plugins I have configured:.*builtin-admin)`, false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "/bender: list disabled plugins", false, []TestMessage{{null, general, `(?s:(Here's a list of all disabled plugins:|There are no disabled plugins))`, false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, ";list jobs", false, []TestMessage{{null, general, `(?s:(Here's a list of jobs for this channel:|I don't see any jobs configured for this channel))`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, ";list all jobs", false, []TestMessage{{null, general, `(?s:(Here's a list of all the jobs I know about:|I dont' have any jobs configured))`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, null, ";show log", false, []TestMessage{{alice, null, ".*", false}}, []bot.Event{bot.BotDirectMessage, bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, null, ";show log page 1", false, []TestMessage{{alice, null, ".*", false}}, []bot.Event{bot.BotDirectMessage, bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestHelp",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, deadzone, ";help", false, []TestMessage{{null, deadzone, `(?s:Quick help.*- /bender help <keyword> - get help for the provided <keyword>.*- /bender help <keyword> brief - compact help for a likely command.*- /bender commands - browse plugins and command groups available in this channel.*- /bender help-all - help for all commands available in this channel, including global commands.*Plugin help: /bender help <plugin>.*Exact command help: /bender help <plugin>/<command>.*Browse this channel: /bender commands)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, deadzone, ";commands", false, []TestMessage{{null, deadzone, `(?s:Plugins and command groups available in this channel.*builtin-help.*Commands: builtin-help/commands.*Help: /bender help builtin-help or /bender help builtin-help/<command>.*Exact help: /bender help <plugin>/<command>.*Search by keyword: /bender help <plugin\|command\|keyword>)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, deadzone, ";help-all", false, []TestMessage{{null, deadzone, `(?s:Commands available in this channel \(including global\).*Command: builtin-help/help.*Usage: help <keyword>.*Exact help: /bender help builtin-help/help)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, deadzone, ";help help", false, []TestMessage{{null, deadzone, `(?s:Help for keyword: help.*Plugin help: help.*Commands:.*- help/help.*Example: ;help with robot.*More detail: /bender help help/help.*Other command matches:.*Command: builtin-help/help.*Exact help: /bender help builtin-help/help)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, deadzone, ";help knock", false, []TestMessage{{null, deadzone, `(?s:Help for keyword: knock.*Plugin help: knock.*Commands:.*- knock/knock - Starts an interactive knock-knock joke\..*Example: ;tell me a knock-knock joke.*More detail: /bender help knock/knock)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, deadzone, ";help knock/knock", false, []TestMessage{{null, deadzone, `(?s:Command help: knock/knock.*Usage: tell me a knock-knock joke.*Availability:)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, ";tell me a jok", false, []TestMessage{{null, general, `(?s:;tell me a jok looks close to knock/knock.*Try: ;tell me a knock-knock joke.*More help: /bender help knock/knock)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
			{aliceID, deadzone, ";knok", false, []TestMessage{{null, deadzone, `(?s:I couldn't match ;knok in channel #deadzone.*#general or #random.*/bender help knock/knock)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestHelpWithoutHiddenCommands",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Capabilities: map[string]robot.ConnectorCapabilities{
			"test": {HiddenCommands: false},
		},
		Cases: legacyCases([]testItem{
			{aliceID, general, ";tell me a jok", false, []TestMessage{{null, general, `(?s:;tell me a jok looks close to knock/knock.*Try: ;tell me a knock-knock joke.*More help: ;help knock/knock)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
			{aliceID, deadzone, ";knok", false, []TestMessage{{null, deadzone, `(?s:I couldn't match ;knok in channel #deadzone.*#general or #random.*;help knock/knock)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestHelpNoAliasWithoutHiddenCommands",
		ConfigDir: "test/membrain-noalias",
		LogName:   "bottest.log",
		Capabilities: map[string]robot.ConnectorCapabilities{
			"test": {HiddenCommands: false},
		},
		Cases: legacyCases([]testItem{
			{aliceID, general, "bender, tell me a jok", false, []TestMessage{{null, general, `(?s:bender, tell me a jok looks close to knock/knock.*Try: bender, tell me a knock-knock joke.*More help: bender, help knock/knock)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
			{aliceID, deadzone, "bender, knok", false, []TestMessage{{null, deadzone, `(?s:I couldn't match bender, knok in channel #deadzone.*#general or #random.*bender, help knock/knock)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestHelpConnectorWithoutProtocolBotNameUsesBotInfoForHiddenHelp",
		ConfigDir: "test/membrain-nameless",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, general, ";tell me a jok", false, []TestMessage{{null, general, `(?s:;tell me a jok looks close to knock/knock.*Try: ;tell me a knock-knock joke.*More help: /bender help knock/knock)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestHelpGroupFiltering",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Flow:      helpGroupFilteringFlow,
	})
}
