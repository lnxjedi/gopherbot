//go:build integration

package bot_test

/*
common_test.go - setup and initialization of "black box" integration testing.

Run integration tests with:
$ go test -v --tags 'test integration' ./test

Run specific tests with e.g.:
$ go test -run MessageMatch -v --tags 'test integration' ./test

To run tests with static builds and modules from vendor:
$ CGO_ENABLED=0 go test -v --tags 'test integration netgo osusergo static_build' -mod vendor ./test

Generate coverage statistics report with:
$ go tool cover -html=coverage.out -o coverage.html

Check status of goroutines if tests get hung up
$ go tool pprof http://localhost:8889/debug/pprof/goroutine
...
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) list lnxjedi
Total: 11
ROUTINE ======================== github.com/lnxjedi/gopherbot/v2/bot...

(eventual) Setup for "clear box" testing of bot internals is in bot_test.go
*/

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/lnxjedi/gopherbot/v2/bot"
	testc "github.com/lnxjedi/gopherbot/v2/connectors/test"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/groups"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/ping"
	_ "github.com/lnxjedi/gopherbot/v2/history/file"

	// Anything referred to robot.yaml has to be compiled in
	_ "github.com/lnxjedi/gopherbot/v2/gojobs/go-bootstrap"

	_ "net/http/pprof"
)

var testInstallPath string

// Environment setting(s) for expanding installed conf/robot.yaml
func init() {
	os.Setenv("GOPHER_PROTOCOL", "test")
	wd, _ := os.Getwd()
	testInstallPath = filepath.Dir(wd)
}

// TestMessage is for sending messages to the robot
type TestMessage struct {
	User, Channel, Message string
	Threaded               bool
}

type testItem struct {
	user, channel, message string
	threaded               bool          // if true the message is sent in a thread
	replies                []TestMessage // note: TestMessage.Message -> regex
	events                 []Event
	pause                  int // time in milliseconds to pause after test item
}

// NOTE: integration tests are closely tied to the configuration in test/...

// Cast of Users
const alice = "alice"
const bob = "bob"
const carol = "carol"
const david = "david"
const erin = "erin"
const aliceID = "u0001"
const bobID = "u0002"
const carolID = "u0003"
const davidID = "u0004"
const erinID = "u0005"

// When the robot doesn't address the user specifically, or sends a DM
const null = ""

// ... and the Channels they play in
const general = "general"
const random = "random"
const bottest = "bottest"
const deadzone = "deadzone"

func setup(cfgdir, logfile string, t *testing.T) (<-chan bool, *testc.TestConnector) {
	os.Setenv("GOPHER_ENCRYPTION_KEY", "gopherbot-integration-tests-brain-key")
	testVer := VersionInfo{"test", "(unknown)"}

	testc.ExportTest.Lock()
	testc.ExportTest.Test = t
	testc.ExportTest.Unlock()

	done, tconn := StartTest(testVer, cfgdir, logfile, t)
	testConnector := tconn.(*testc.TestConnector)

	return done, testConnector
}

func teardown(t *testing.T, done <-chan bool, conn *testc.TestConnector) {
	// Alice is a bot admin who can order the bot to quit in #general
	conn.SendBotMessage(&testc.TestMessage{aliceID, null, "quit", false, false})

	// Now we wait for the connection to finish
	<-done
	ws := filepath.Join(testInstallPath, "test", "workspace")
	if err := os.RemoveAll(ws); err != nil {
		fmt.Printf("Removing temporary workspace: %v\n", err)
	}

	evOk := true
	ev := GetEvents()
	want := []Event{BotDirectMessage, AdminCheckPassed, CommandTaskRan, GoPluginRan}
	if len(*ev) != len(want) {
		evOk = false
	} else {
		for i, e := range *ev {
			if e != want[i] {
				evOk = false
			}
		}
	}
	if !evOk {
		gevs := make([]string, len(*ev))
		for i, e := range *ev {
			gevs[i] = e.String()
		}
		wevs := make([]string, len(want))
		for i, e := range want {
			wevs[i] = e.String()
		}
		t.Errorf("FAILED teardown events; want: \"%s\"; got: %s\n", strings.Join(wevs, ", "), strings.Join(gevs, ", "))
	}
}

func testcases(t *testing.T, conn *testc.TestConnector, tests []testItem) {
	for _, test := range tests {
		// Clear out start-up events
		GetEvents()
		hidden := false
		message := test.message
		if strings.HasPrefix(message, "/") {
			hidden = true
			message = strings.TrimPrefix(message, "/")
		}
		conn.SendBotMessage(&testc.TestMessage{test.user, test.channel, message, test.threaded, hidden})
		for _, want := range test.replies {
			if re, err := regexp.Compile(want.Message); err != nil {
				t.Errorf("FAILED: regex \"%s\" didn't compile: %v", want.Message, err)
			} else {
				got, err := conn.GetBotMessage()
				if err != nil {
					t.Errorf("FAILED timeout waiting for reply from robot; want: \"%s\"", want.Message)
				} else {
					if !re.MatchString(got.Message) {
						t.Errorf("FAILED message regex match; want: \"%s\", got: \"%s\"", want.Message, got.Message)
					} else {
						if got.User != want.User || got.Channel != want.Channel || got.Threaded != want.Threaded {
							t.Errorf("FAILED user/channel match; want u:%s,c:%s,t:%t; got u:%s,c:%s,t:%t", want.User, want.Channel, want.Threaded, got.User, got.Channel, got.Threaded)
						}
					}
				}
			}
		}
		ev := GetEvents()
		evOk := true
		if len(*ev) != len(test.events) {
			evOk = false
		} else {
			for i, e := range *ev {
				if e != test.events[i] {
					evOk = false
				}
			}
		}
		if !evOk {
			wevs := make([]string, len(test.events))
			for i, e := range test.events {
				wevs[i] = e.String()
			}
			gevs := make([]string, len(*ev))
			for i, e := range *ev {
				gevs[i] = e.String()
			}
			t.Errorf("FAILED emitted events; want: \"%s\"; got: %s\n", strings.Join(wevs, ", "), strings.Join(gevs, ", "))
		}
		if test.pause > 0 {
			time.Sleep(time.Millisecond * time.Duration(test.pause))
		}
	}
}
