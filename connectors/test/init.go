package test

import (
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

// Global persistent map of user name to user index
var userIDMap = make(map[string]int)
var userMap = make(map[string]int)

func rebuildUserIndexes(users []testUser) {
	nextUserIDMap := make(map[string]int, len(users))
	nextUserMap := make(map[string]int, len(users))
	for i, u := range users {
		nextUserIDMap[u.InternalID] = i
		nextUserMap[u.Name] = i
	}
	userIDMap = nextUserIDMap
	userMap = nextUserMap
}

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
	Users    []testUser
	Channels []string
}

func init() {
	robot.RegisterConnector("test", Initialize)
}

// Initialize sets up the connector and returns a connector object
func Initialize(handler robot.Handler, l *log.Logger) robot.InitializedConnector {
	var c config

	err := handler.GetProtocolConfig(&c)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to retrieve protocol configuration: %v", err)
	}
	botInfo := handler.GetBotInfo()
	botName := strings.TrimSpace(botInfo.UserName)
	if botName == "" {
		botName = "gopherbot"
	}
	botFullName := strings.TrimSpace(botInfo.FullName)
	if botFullName == "" {
		botFullName = botName
	}

	rebuildUserIndexes(c.Users)

	ExportTest.Lock()
	t := ExportTest.Test
	ExportTest.Unlock()

	tc := &TestConnector{
		botName:     botName,
		botFullName: botFullName,
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

	return robot.InitializedConnector{
		Connector:    robot.Connector(tc),
		Capabilities: robot.ConnectorCapabilities{HiddenCommands: true},
	}
}
