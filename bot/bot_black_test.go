package bot_test

/*
bot_test.go - setup and initialization of "black box" plugin testing.

Setup for "clear box" testing of bot internals is in bot/bot_test.go
*/

import (
	"fmt"
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

func teardown(done <-chan struct{}, testConnector *testc.TestConnector) {
	// Alice is a bot admin who can order the bot to quit in #general
	testConnector.Listener <- "|ualice"
	testConnector.Listener <- "|cgeneral"
	testConnector.Listener <- ";quit"

	// Now we wait for the connection to finish
	<-done
}

func TestPing(t *testing.T) {
	done, conn := setup("cfg/test", t)
	fmt.Printf("Starting test ping with connector: %p and listener: %v\n", conn, conn.Listener)

	conn.SendBotMessage(&TestMessage{alice, general, "ping"})
	reply = conn.GetBotMessage()
	t.Logf("Reply from robot: %v", reply)

	teardown(done, conn)
}
