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

func setup(t *testing.T) (<-chan struct{}, *testc.TestConnector) {
	done, tconn := StartTest(t)
	testConnector := tconn.(*testc.TestConnector)
	testConnector.Test = t

	return done, testConnector
}

// TODO: fix teardown - the background goroutine is draining ALL messages,
// and not letting the previous connetion finish

func teardown(done <-chan struct{}, testConnector *testc.TestConnector) {
	// Alice is a bot admin who can order the bot to quit in #general
	testConnector.Listener <- "|ualice"
	testConnector.Listener <- "|cgeneral"
	testConnector.Listener <- ";quit"

	// Now we wait for the connection to finish
	<-done
}

func send(t *testing.T, conn *testc.TestConnector, msg string) {
	t.Logf("Sending to robot: %s", msg)
	conn.Listener <- msg
}

func TestPing(t *testing.T) {
	done, conn := setup(t)
	fmt.Printf("Starting test ping with connector: %p and listener: %v\n", conn, conn.Listener)

	var reply string

	send(t, conn, ";ping")
	reply = <-conn.Speaking
	t.Logf("Reply from robot: %s", reply)

	send(t, conn, ";ping")
	reply = <-conn.Speaking
	t.Logf("Reply from robot: %s", reply)

	teardown(done, conn)
}

func TestPing2(t *testing.T) {
	done, conn := setup(t)
	fmt.Printf("Starting test ping with connector: %p and listener: %v\n", conn, conn.Listener)

	var reply string

	send(t, conn, ";ping")
	reply = <-conn.Speaking
	t.Logf("Reply from robot: %s", reply)

	send(t, conn, ";ping")
	reply = <-conn.Speaking
	t.Logf("Reply from robot: %s", reply)

	teardown(done, conn)
}
