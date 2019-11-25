package test

import (
	"log"
	"sync"
	"testing"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/lnxjedi/gopherbot/robot"
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
func Initialize(handler robot.Handler, l *log.Logger) robot.Connector {
	var c config

	err := handler.GetProtocolConfig(&c)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to retrieve protocol configuration: %v", err)
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

	tc.Handler = handler
	tc.SetBotID(tc.botID)
	tc.Log(robot.Info, "Set bot ID to", tc.botID)

	return robot.Connector(tc)
}
