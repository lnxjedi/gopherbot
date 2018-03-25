package bot_test

/*
bot_test.go - setup and initialization of "black box" plugin testing.

Setup for "clear box" testing of bot internals is in bot/bot_test.go
*/

import (
	_ "net/http/pprof"
	"testing"

	. "github.com/lnxjedi/gopherbot/bot"
	_ "github.com/lnxjedi/gopherbot/brains/mem"
	testc "github.com/lnxjedi/gopherbot/connectors/test"
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/links"
	_ "github.com/lnxjedi/gopherbot/goplugins/lists"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"
)

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

func setup(cfgdir string, t *testing.T) (<-chan struct{}, *testc.TestConnector) {
	done, tconn := StartTest(cfgdir, t)
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

func TestPing(t *testing.T) {
	done, conn := setup("cfg/test", t)

	conn.SendBotMessage(&testc.TestMessage{alice, general, ";ping"})
	reply := conn.GetBotMessage()
	if reply.Message != "PONG" {
		t.Errorf("Wrong reply: %s", reply.Message)
	}
	ev := GetEvents()
	t.Logf("Got events: %v", ev)

	teardown(done, conn)
}

func TestPing2(t *testing.T) {
	done, conn := setup("cfg/test", t)

	conn.SendBotMessage(&testc.TestMessage{alice, general, ";ping"})
	reply := conn.GetBotMessage()
	if reply.Message != "PONG" {
		t.Errorf("Wrong reply: %s", reply.Message)
	}

	teardown(done, conn)
}
