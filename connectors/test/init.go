package test

import (
	"fmt"
	"log"

	"github.com/lnxjedi/gopherbot/bot"
)

// Global persistent map of user name to user index
var userMap = make(map[string]int)

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
		robot.Log(bot.Fatal, fmt.Errorf("Unable to retrieve protocol configuration: %v", err))
	}

	for i, u := range c.Users {
		userMap[u.Name] = i
	}

	tc := &TestConnector{
		botName:     c.BotName,
		botFullName: c.BotFullName,
		botID:       "deadbeef", // yes - hex in a string
		users:       c.Users,
		channels:    c.Channels,
		listener:    make(chan *TestMessage),
		speaking:    make(chan *TestMessage),
	}

	tc.Handler = robot
	tc.SetFullName(tc.botFullName)
	tc.Log(bot.Debug, "Set bot full name to", tc.botFullName)
	tc.SetName(tc.botName)
	tc.Log(bot.Info, "Set bot name to", tc.botName)

	return bot.Connector(tc)
}
