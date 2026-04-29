package suites

import (
	"context"
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/bot"
)

func init() {
	Register(Suite{
		Name:      "TestBotNameHiddenCommandsUnsupportedConnector",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Capabilities: map[string]robot.ConnectorCapabilities{
			"test": {HiddenCommands: false},
		},
		Cases: legacyCases([]testItem{
			{aliceID, null, "/ping", false, []TestMessage{{alice, null, "This command isn't supported with test because hidden commands are unavailable for this connector\\. Check with the robot administrator\\.", false}}, []bot.Event{bot.BotDirectMessage}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestBotNoName",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, null, ";ping", false, []TestMessage{{alice, null, "PONG", false}}, []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, null, "ping", false, []TestMessage{{alice, null, "PONG", false}}, []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, ";ping", false, []TestMessage{{alice, general, "PONG", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "ping;", false, []TestMessage{}, []bot.Event{}, 0},
			{bobID, general, "bender: echo hello world", false, []TestMessage{{null, general, "hello world", true}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, "ping", false, []TestMessage{}, []bot.Event{}, 500},
			{aliceID, general, ";", false, []TestMessage{{alice, general, "PONG", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "ping", false, []TestMessage{}, []bot.Event{}, 100},
			{aliceID, general, "hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []bot.Event{bot.AmbientTaskRan, bot.ExternalTaskRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestBotNoAlias",
		ConfigDir: "test/membrain-noalias",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, null, "ping, bender", false, []TestMessage{{alice, null, "PONG", false}}, []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, null, "bender ping", false, []TestMessage{{alice, null, "PONG", false}}, []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, null, "ping", false, []TestMessage{{alice, null, "PONG", false}}, []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "ping, bender", false, []TestMessage{{alice, general, "PONG", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "bender ping", false, []TestMessage{{alice, general, "PONG", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "bender, ping", false, []TestMessage{{alice, general, "PONG", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "@bender ping", false, []TestMessage{{alice, general, "PONG", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "ping, @bender", false, []TestMessage{{alice, general, "PONG", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{bobID, general, "bender: echo hello world", false, []TestMessage{{null, general, "hello world", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, "ping", false, []TestMessage{}, []bot.Event{}, 200},
			{aliceID, general, "bender", false, []TestMessage{{alice, general, "PONG", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "ping", false, []TestMessage{}, []bot.Event{}, 100},
			{aliceID, general, "hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []bot.Event{bot.AmbientTaskRan, bot.ExternalTaskRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestReload",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, general, "reload, bender", false, []TestMessage{{null, general, "Starting init job 'go-bootstrap'.*", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan, bot.ScheduledTaskRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestMessageMatch",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, general, "hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []bot.Event{bot.AmbientTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, ";hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []bot.Event{bot.AmbientTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, ";hello world", false, []TestMessage{{null, general, "Hello, World!", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, ";HELLO   WORLD", false, []TestMessage{{null, general, "Hello, World!", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, ";hello-world", false, []TestMessage{{null, general, "Hello, World!", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, null, "hello robot", false, []TestMessage{{alice, null, "Hello, World!", false}}, []bot.Event{bot.BotDirectMessage, bot.AmbientTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, null, "bender, hello robot", false, []TestMessage{{alice, null, "Hello, World!", false}}, []bot.Event{bot.BotDirectMessage, bot.AmbientTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, "ping", false, []TestMessage{}, []bot.Event{}, 100},
			{aliceID, general, ";hello robot", false, []TestMessage{{null, general, "Hello, World!", false}}, []bot.Event{bot.AmbientTaskRan, bot.ExternalTaskRan}, 100},
			{aliceID, general, "bender", false, []TestMessage{{null, general, `Yes\?`, false}}, []bot.Event{}, 0},
			{aliceID, random, "hello robot", false, []TestMessage{{null, random, "Hello, World!", false}}, []bot.Event{bot.AmbientTaskRan, bot.ExternalTaskRan}, 100},
			{aliceID, random, ";hello robot", false, []TestMessage{{null, random, "I'm here", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestVisibility",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, general, "help ruby, bender", false, []TestMessage{{null, general, `(?s:Command matches for keyword: ruby.*Availability: channels: random)`, true}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "ruby me, bender", false, []TestMessage{{null, general, "rubydemo/ruby not available in #general, try #random", true}}, []bot.Event{}, 0},
			{aliceID, deadzone, "bender: echo hello world", false, []TestMessage{{null, deadzone, "echo/echo not available in #deadzone, try one of: #general, #random", true}}, []bot.Event{}, 0},
			{aliceID, null, "hear me out", false, []TestMessage{{alice, null, "bashdemo/hear not available in direct messages, try it in any regular channel", false}}, []bot.Event{bot.BotDirectMessage}, 0},
			{bobID, general, ";ping", false, []TestMessage{{null, general, `(?s:I couldn't match .*More help: .*help builtin-help/help.*Try .*commands.*help <keyword>.*)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
			{bobID, general, ";reload", false, []TestMessage{{null, general, `(?s:I couldn't match .*Try .*commands.*help <keyword>.*)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestValidateUser",
		ConfigDir: "test/membrain",
		LogName:   "bottest-validate-user.log",
		Flow:      validateUserFlow,
	})
	Register(Suite{
		Name:      "TestPrompting",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{carolID, general, "Bender, listen to me", false, []TestMessage{{carol, null, "Ok, .*", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{carolID, null, "You're pretty cool", false, []TestMessage{{carol, null, "I hear .*cool\"", false}}, []bot.Event{bot.BotDirectMessage}, 0},
			{bobID, general, "hear me out, Bender", false, []TestMessage{{bob, general, "Well ok then.*", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{bobID, general, "I like kittens", false, []TestMessage{{bob, general, "Ok, I hear you saying \"I like kittens\".*", false}}, []bot.Event{}, 0},
			{davidID, general, ";waitask", false, []TestMessage{}, []bot.Event{}, 200},
			{davidID, general, ";asknow", false, []TestMessage{{david, general, `Do you like puppies\?`, false}, {null, general, `ok - answer puppies`, false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{davidID, general, "yes", false, []TestMessage{{david, general, `Do you like kittens\?`, false}, {null, general, `I like puppies too!`, false}}, []bot.Event{}, 0},
			{davidID, general, "yes", false, []TestMessage{{null, general, `I like kittens too!`, false}}, []bot.Event{}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestFormatting",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, general, ";format fixed", false, []TestMessage{{null, general, "_ITALICS_ <ONE> \\*BOLD\\* `CODE` @PARSLEY", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, ";format variable", false, []TestMessage{{null, general, "_italics_ <one> \\*bold\\* `code` @parsley", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
			{aliceID, general, ";format raw", false, []TestMessage{{null, general, "_Italics_ <One> \\*Bold\\* `Code` @parsley", false}}, []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}, 0},
		}),
	})
	Register(Suite{
		Name:      "TestDevel",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: legacyCases([]testItem{
			{aliceID, general, ";create a new grocery list", false, []TestMessage{{null, general, `(?s:I couldn't match .*create a new grocery list.*More help: /bender help lists/add.*Try /bender commands or /bender help <keyword>\.)`, true}}, []bot.Event{bot.CatchAllsRan, bot.CatchAllTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, ";add bananas to the grocery list", false, []TestMessage{{alice, general, "I don't have a 'grocery' list, do you want to create it?", false}}, []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}, 0},
			{aliceID, general, "yes", false, []TestMessage{{null, general, "Ok, I created a new grocery list and added bananas to it", false}}, []bot.Event{}, 0},
		}),
	})
}

func validateUserFlow(ctx context.Context, d Driver) []Failure {
	const suiteName = "TestValidateUser"
	failures := make([]Failure, 0)
	got, err := sendAndReceive(ctx, d, Message{User: aliceID, Text: "validate user bob"}, ExpectedMessage{User: alice, TextPattern: `Validation code for 'bob': ([0-9]{7})`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "validation-code", "reply", "%v", err)
		return failures
	}
	codeRe := regexp.MustCompile(`Validation code for 'bob': ([0-9]{7})`)
	matches := codeRe.FindStringSubmatch(got.Text)
	if len(matches) != 2 {
		addFlowFailure(&failures, suiteName, "validation-code", "extract", "validation code reply = %q", got.Text)
		return failures
	}
	_, _ = d.DrainEvents(ctx)
	got, err = sendAndReceive(ctx, d, Message{User: bobID, Text: matches[1]}, ExpectedMessage{User: alice, TextPattern: "User validation received: test user 'bob' has internal ID 'u0002'"})
	if err != nil {
		addFlowFailure(&failures, suiteName, "validation-notify", "reply", "%v", err)
		return failures
	}
	if got.Text != "User validation received: test user 'bob' has internal ID 'u0002'" {
		addFlowFailure(&failures, suiteName, "validation-notify", "reply", "validation notification = %q", got.Text)
	}
	_, _ = d.DrainEvents(ctx)
	return failures
}

func helpGroupFilteringFlow(ctx context.Context, d Driver) []Failure {
	const suiteName = "TestHelpGroupFiltering"
	failures := make([]Failure, 0)
	getHelpReply := func(user, caseName string) string {
		got, err := sendAndReceive(ctx, d, Message{User: user, Channel: general, Text: ";help lists"}, ExpectedMessage{Channel: general, TextPattern: `(?s:.*)`, Threaded: true})
		if err != nil {
			addFlowFailure(&failures, suiteName, caseName, "reply", "%v", err)
			return ""
		}
		_, _ = d.DrainEvents(ctx)
		return got.Text
	}
	aliceHelp := getHelpReply(aliceID, "alice-help")
	if aliceHelp != "" {
		if !strings.Contains(aliceHelp, "lists/add") {
			addFlowFailure(&failures, suiteName, "alice-help", "content", "expected lists help output for alice to include at least one lists command, got: %q", aliceHelp)
		}
		if strings.Contains(aliceHelp, "lists/send") {
			addFlowFailure(&failures, suiteName, "alice-help", "content", "expected help to hide unauthorized command '[lists] send' for alice, got: %q", aliceHelp)
		}
	}
	bobHelp := getHelpReply(bobID, "bob-help")
	if bobHelp != "" && !strings.Contains(bobHelp, "lists/send") {
		addFlowFailure(&failures, suiteName, "bob-help", "content", "expected help to include authorized command '[lists] send' for bob, got: %q", bobHelp)
	}
	return failures
}
