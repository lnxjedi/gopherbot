// +build integration

package bot_test

/*
bot_integration_test.go - setup and initialization of "black box" integration testing.

Setup for "clear box" testing of bot internals is in bot_test.go
*/

import (
	_ "net/http/pprof"
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

// Cast of Users
const alice = "alice"
const bob = "bob"
const carol = "carol"
const david = "david"
const erin = "erin"

// ... and the Channels the play in
const general = "general"
const random = "random"
const bottest = "bottest"

func setup(cfgdir, logfile string, t *testing.T) (<-chan struct{}, *testc.TestConnector) {
	done, tconn := StartTest(cfgdir, logfile, t)
	testConnector := tconn.(*testc.TestConnector)
	testConnector.SetTest(t)

	return done, testConnector
}

func teardown(done <-chan struct{}, conn *testc.TestConnector) {
	// Alice is a bot admin who can order the bot to quit in #general
	conn.SendBotMessage(&testc.TestMessage{alice, general, ";quit"})

	// Now we wait for the connection to finish
	<-done
}

func testcases(t *testing.T, conn *testc.TestConnector, tests []testItem) {
	for _, test := range tests {
		conn.SendBotMessage(&testc.TestMessage{test.user, test.channel, test.message})
		for _, want := range test.replies {
			got := conn.GetBotMessage() // TODO: match message based on regex for want
			if got.User != want.User || got.Channel != want.Channel || got.Message != want.Message {
				t.Errorf("FAILED: want u:%s, c:%s, m:%s; got u:%s,c:%s,m:%s", want.User, want.Channel, want.Message, got.User, got.Channel, got.Message)
			}
		}
	}
}

func TestBotName(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "test.log", t)

	tests := []testItem{
		{alice, general, ";ping", []testc.TestMessage{{alice, general, "PONG"}}, []Event{CommandPluginRan, GoPluginRan}},
	}
	testcases(t, conn, tests)

	teardown(done, conn)
}

func TestPing(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "test.log", t)

	conn.SendBotMessage(&testc.TestMessage{alice, general, ";ping"})
	reply := conn.GetBotMessage()
	if reply.Message != "PONG" {
		t.Errorf("FAILED - Wrong reply: %s", reply.Message)
	}
	ev := GetEvents()
	t.Logf("Got events: %v", ev)

	teardown(done, conn)
}

func TestReload(t *testing.T) {
	done, conn := setup("cfg/test/filebrain", "test.log", t)

	conn.SendBotMessage(&testc.TestMessage{alice, general, ";reload"})
	reply := conn.GetBotMessage()
	if reply.Message != "Configuration reloaded successfully" {
		t.Errorf("FAILED - Wrong reply: %s", reply.Message)
	}
	ev := GetEvents()
	t.Logf("Got events: %v", ev)

	teardown(done, conn)
}
