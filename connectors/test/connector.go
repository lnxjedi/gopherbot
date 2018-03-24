// Package test implements a test connector for automated testing.

package test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/lnxjedi/gopherbot/bot"
)

// TestConnector holds all the relevant data about a connection
type TestConnector struct {
	currentChannel string      // The current channel for the user
	currentUser    string      // The current userid
	channels       []string    // the channels the robot is in
	botName        string      // human-readable name of bot
	botFullName    string      // human-readble full name of the bot
	botID          string      // slack internal bot ID
	users          []testUser  // configured users
	Listener       chan string // input channel for test functions to send messages from a user
	Speaking       chan string // output channel for test functions to get messages from the bot
	Test           *testing.T  // for the connector to log
	bot.Handler                // bot API for connectors
	sync.RWMutex               // shared mutex for locking connector data structures
}

func (tc *TestConnector) Run(stop <-chan struct{}) {

loop:
	for {

		select {
		case <-stop:
			tc.Test.Log(bot.Debug, "Received stop in connector")
			break loop
		case input := <-tc.Listener:
			if len(input) == 0 {
				continue
			}
			if input[0] == '|' {
				if len(input) == 1 {
					continue
				}
				switch input[1] {
				case 'C', 'c':
					exists := false
					newchan := input[2:]
					tc.Lock()
					if newchan == "" {
						tc.currentChannel = ""
					} else {
						for _, ch := range tc.channels {
							if ch == newchan {
								exists = true
								break
							}
						}
						if exists {
							tc.currentChannel = newchan
							tc.Test.Log(bot.Info, fmt.Sprintf("Changed current channel to: %s", newchan))
						} else {
							tc.Test.Log(bot.Fatal, "Invalid channel.")
						}
					}
					tc.Unlock()
				case 'U', 'u':
					exists := false
					newuser := input[2:]
					tc.Lock()
					if newuser == "" {
						tc.Test.Log(bot.Fatal, "Invalid 0-length user")
					} else {
						for _, u := range tc.users {
							if u.Name == newuser {
								exists = true
							}
						}
						if exists {
							tc.currentUser = newuser
							tc.Test.Log(bot.Info, fmt.Sprintf("Changed current user to: %s", newuser))
						} else {
							tc.Test.Log(bot.Fatal, "Invalid user.")
						}
					}
					tc.Unlock()
				default:
					tc.Test.Log(bot.Fatal, "Invalid connector command")
				}
			} else {
				tc.RLock()
				tc.IncomingMessage(tc.currentChannel, tc.currentUser, input)
				tc.RUnlock()
			}
		}
	}
}

// Public 'bot methods all call sendMessage to send a message to a user/channel
func (tc *TestConnector) sendMessage(ch, msg string) (ret bot.RetVal) {
	found := false
	tc.RLock()
	if strings.HasPrefix(ch, "(dm:") {
		found = true
	} else {
		for _, channel := range tc.channels {
			if channel == ch {
				found = true
				break
			}
		}
	}
	tc.RUnlock()
	if !found {
		tc.Test.Log(bot.Error, "Channel not found:", ch)
		return bot.ChannelNotFound
	}
	tc.Speaking <- fmt.Sprintf("%s: %s\n", ch, msg)
	return bot.Ok
}
