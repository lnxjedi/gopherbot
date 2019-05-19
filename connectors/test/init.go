package test

import (
	"fmt"
	"log"
	"sync"
	"testing"

	"github.com/lnxjedi/gopherbot/bot"
)

// Global persistent map of user name to user index
var userIDMap = make(map[string]int)
var userMap = make(map[string]int)

// ExportTest lets bot_integration_test safely supply the *testing.T
var ExportTest = struct {
	Test *testing.T
	sync.Mutex
}{}

type testUser struct {
	Name                                        string // username / handle
	InternalID                                  string // connector internal identifier
	Email, FullName, FirstName, LastName, Phone string
}

type config struct {
	BotName     string // the short name used for addressing the robot
	BotFullName string // the full name of the bot
	Users       []testUser
	Channels    []string
}

func init() {
	bot.RegisterConnector("test", Initialize)
}

// Initialize sets up the connector and returns a connector object
func Initialize(robot bot.Handler, l *log.Logger) bot.Connector {
	var c config

	err := robot.GetProtocolConfig(&c)
	if err != nil {
		robot.Log(bot.Fatal, "Unable to retrieve protocol configuration: %v", err)
	}

	for i, u := range c.Users {
		userIDMap[u.InternalID] = i
		userMap[u.Name] = i
	}

	ExportTest.Lock()
	t := ExportTest.Test
	ExportTest.Unlock()

	tc := &TestConnector{
		botName:     c.BotName,
		botFullName: c.BotFullName,
		botID:       "deadbeef", // yes - hex in a string
		users:       c.Users,
		channels:    c.Channels,
		listener:    make(chan *TestMessage),
		speaking:    make(chan *TestMessage),
		test:        t,
	}

	tc.Handler = robot
	tc.SetBotIDtc.botID)
	tc.Log(bot.Info, "Set bot ID to", tc.botID)

	return bot.Connector(tc)
}
