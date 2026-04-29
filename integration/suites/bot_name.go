package suites

import (
	"time"

	"github.com/lnxjedi/gopherbot/v2/bot"
)

func init() {
	Register(Suite{
		Name:      "TestBotName",
		ConfigDir: "test/membrain",
		LogName:   "bottest.log",
		Cases: []Case{
			{Input: Message{User: AliceID, Text: "ping, bender"}, Replies: []ExpectedMessage{{User: Alice, TextPattern: "PONG"}}, Events: []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Text: ";ping"}, Replies: []ExpectedMessage{{User: Alice, TextPattern: "PONG"}}, Events: []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Text: "/ping"}, Replies: []ExpectedMessage{{User: Alice, TextPattern: "\\(Use /bender <command> to address a hidden command\\.\\)"}}, Events: []bot.Event{bot.BotDirectMessage}},
			{Input: Message{User: AliceID, Text: "bender ping"}, Replies: []ExpectedMessage{{User: Alice, TextPattern: "PONG"}}, Events: []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Text: "ping"}, Replies: []ExpectedMessage{{User: Alice, TextPattern: "PONG"}}, Events: []bot.Event{bot.BotDirectMessage, bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Channel: General, Text: "ping, bender"}, Replies: []ExpectedMessage{{User: Alice, Channel: General, TextPattern: "PONG"}}, Events: []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Channel: General, Text: ";ping"}, Replies: []ExpectedMessage{{User: Alice, Channel: General, TextPattern: "PONG"}}, Events: []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Channel: General, Text: "bender ping"}, Replies: []ExpectedMessage{{User: Alice, Channel: General, TextPattern: "PONG"}}, Events: []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Channel: General, Text: "bender, ping"}, Replies: []ExpectedMessage{{User: Alice, Channel: General, TextPattern: "PONG"}}, Events: []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Channel: General, Text: "@bender ping"}, Replies: []ExpectedMessage{{User: Alice, Channel: General, TextPattern: "PONG"}}, Events: []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Channel: General, Text: "ping, @bender"}, Replies: []ExpectedMessage{{User: Alice, Channel: General, TextPattern: "PONG"}}, Events: []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Channel: General, Text: "ping;"}, Events: []bot.Event{}},
			{Input: Message{User: BobID, Channel: General, Text: "bender: echo hello world"}, Replies: []ExpectedMessage{{Channel: General, TextPattern: "Sure thing: hello world", Threaded: true}}, Events: []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}},
			{Input: Message{User: BobID, Channel: General, Text: "/bender: echo hello world"}, Replies: []ExpectedMessage{{Channel: General, TextPattern: "\\(Sure thing: hello world\\)", Threaded: true}}, Events: []bot.Event{bot.CommandTaskRan, bot.ExternalTaskRan}},
			{Input: Message{User: AliceID, Channel: General, Text: "ping"}, Events: []bot.Event{}, Pause: 300 * time.Millisecond},
			{Input: Message{User: AliceID, Channel: General, Text: "bender"}, Replies: []ExpectedMessage{{User: Alice, Channel: General, TextPattern: "PONG"}}, Events: []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}},
			{Input: Message{User: AliceID, Channel: General, Text: "ping"}, Events: []bot.Event{}, Pause: 300 * time.Millisecond},
			{Input: Message{User: AliceID, Channel: General, Text: ";"}, Replies: []ExpectedMessage{{User: Alice, Channel: General, TextPattern: "PONG"}}, Events: []bot.Event{bot.CommandTaskRan, bot.GoPluginRan}},
		},
	})
}
