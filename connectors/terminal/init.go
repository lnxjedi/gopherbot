package terminal

import (
	"fmt"
	"log"
	"os"
	"path"
	"sync"

	"github.com/chzyer/readline"
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
	Channels     []string
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
	found := false
	for i, u := range c.Users {
		userMap[u.Name] = i
		if c.StartUser == u.Name {
			found = true
		}
	}
	if !found {
		robot.Log(bot.Fatal, fmt.Sprintf("Start user \"%s\" not listed in Users array", c.StartUser))
	}

	found = false
	for _, ch := range c.Channels {
		if c.StartChannel == ch {
			found = true
		}
	}
	if !found {
		robot.Log(bot.Fatal, fmt.Sprintf("Start channel \"%s\" not listed in Channels array", c.StartChannel))
	}

	var histfile string
	home := os.Getenv("HOME")
	if len(home) == 0 {
		home = os.Getenv("USERPROFILE")
	}
	if len(home) > 0 {
		histfile = path.Join(home, ".gopherbot_history")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            fmt.Sprintf("c:%s/u:%s -> ", c.StartChannel, c.StartUser),
		HistoryFile:       histfile,
		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}

	if !robot.GetLogToFile() {
		l.SetOutput(rl.Stdout())
	}

	tc := &termConnector{
		currentChannel: c.StartChannel,
		currentUser:    c.StartUser,
		channels:       c.Channels,
		running:        false,
		botName:        c.BotName,
		botFullName:    c.BotFullName,
		botID:          "deadbeef", // yes - hex in a string
		users:          c.Users,
		heard:          make(chan string),
		reader:         rl,
	}

	tc.Handler = robot
	tc.SetFullName(tc.botFullName)
	tc.Log(bot.Debug, "Set bot full name to", tc.botFullName)
	tc.SetName(tc.botName)
	tc.Log(bot.Info, "Set bot name to", tc.botName)

	return bot.Connector(tc)
}
