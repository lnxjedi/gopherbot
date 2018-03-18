package terminal

import (
	"fmt"
	"log"
	"sync"

	"github.com/lnxjedi/gopherbot/bot"
)

// Global persistent map of user name to user index
var userMap = make(map[string]int)

type termUser struct {
	Name                                        string // username / handle
	InternalID                                  string // connector internal identifier
	Email, FullName, FirstName, LastName, Phone string
}

type config struct {
	StartChannel string // the initial channel
	StartUser    string // the initial userid
	BotName      string // the short name used for addressing the robot
	BotFullName  string // the full name of the bot
	Users        []termUser
}

var lock sync.Mutex // package var lock
var started bool    // set when connector is started

func init() {
	bot.RegisterConnector("term", Initialize)
}

// Initialize sets up the connector and returns a connector object
func Initialize(robot bot.Handler, l *log.Logger) bot.Connector {
	lock.Lock()
	if started {
		lock.Unlock()
		return nil
	}
	started = true
	lock.Unlock()

	var c config

	err := robot.GetProtocolConfig(&c)
	if err != nil {
		robot.Log(bot.Fatal, fmt.Errorf("Unable to retrieve protocol configuration: %v", err))
	}
	for i, u := range c.Users {
		userMap[u.Name] = i
	}

	tc := &termConnector{
		currentChannel: c.StartChannel,
		currentUser:    c.StartUser,
		channels:       make([]string, 0),
		running:        false,
		botName:        c.BotName,
		botFullName:    c.BotFullName,
		botID:          "deadbeef",
		users:          c.Users,
		heard:          make(chan string),
		speaking:       make(chan struct{}),
	}

	tc.Handler = robot
	tc.SetFullName(tc.botFullName)
	tc.Log(bot.Debug, "Set bot full name to", tc.botFullName)
	tc.SetName(tc.botName)
	tc.Log(bot.Info, "Set bot name to", tc.botName)

	return bot.Connector(tc)
}
