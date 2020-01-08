package terminal

import (
	"fmt"
	"log"
	"os"
	"path"
	"sync"

	"github.com/chzyer/readline"
	"github.com/lnxjedi/gopherbot/robot"
)

// Global persistent map of user name to user index
var userIDMap = make(map[string]int)
var userMap = make(map[string]int)

type termUser struct {
	Name                                        string // username / handle
	InternalID                                  string // connector internal identifier
	Email, FullName, FirstName, LastName, Phone string
}

type config struct {
	StartChannel string // the initial channel
	StartUser    string // the initial userid
	EOF          string // command to send on EOF (ctrl-D), default ";quit"
	Users        []termUser
	Channels     []string
}

var lock sync.Mutex // package var lock
var started bool    // set when connector is started

// Initialize sets up the connector and returns a connector object
func Initialize(handler robot.Handler, l *log.Logger) robot.Connector {
	lock.Lock()
	if started {
		lock.Unlock()
		return nil
	}
	started = true
	lock.Unlock()

	var c config

	err := handler.GetProtocolConfig(&c)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to retrieve protocol configuration: %v", err)
	}
	eof := ";quit"
	if len(c.EOF) != 0 {
		eof = c.EOF
	}
	found := false
	for i, u := range c.Users {
		userMap[u.Name] = i
		userIDMap[u.InternalID] = i
		if c.StartUser == u.Name {
			found = true
		}
	}
	if !found {
		handler.Log(robot.Fatal, "Start user \"%s\" not listed in Users array", c.StartUser)
	}

	found = false
	for _, ch := range c.Channels {
		if c.StartChannel == ch {
			found = true
		}
	}
	if !found {
		handler.Log(robot.Fatal, "Start channel \"%s\" not listed in Channels array", c.StartChannel)
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
		EOFPrompt:         "exit",
	})
	if err != nil {
		panic(err)
	}

	tc := &termConnector{
		currentChannel: c.StartChannel,
		currentUser:    c.StartUser,
		eof:            eof,
		channels:       c.Channels,
		running:        false,
		users:          c.Users,
		heard:          make(chan string),
		reader:         rl,
	}

	tc.Handler = handler
	tc.SetTerminalWriter(tc.reader)
	return robot.Connector(tc)
}
