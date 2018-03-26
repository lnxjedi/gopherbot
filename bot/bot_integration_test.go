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

func TestPing(t *testing.T) {
	done, conn := setup("cfg/test/membrain", "", t)

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
	done, conn := setup("cfg/test/filebrain", "", t)

	conn.SendBotMessage(&testc.TestMessage{alice, general, ";reload"})
	reply := conn.GetBotMessage()
	if reply.Message != "Configuration reloaded successfully" {
		t.Errorf("FAILED - Wrong reply: %s", reply.Message)
	}
	ev := GetEvents()
	t.Logf("Got events: %v", ev)

	teardown(done, conn)
}
