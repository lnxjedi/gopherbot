// Package test implements a test connector for automated testing.

package test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/lnxjedi/gopherbot/bot"
)

// testConnector holds all the relevant data about a connection
type testConnector struct {
	currentChannel string      // The current channel for the user
	currentUser    string      // The current userid
	channels       []string    // the channels the robot is in
	running        bool        // set on call to Run
	botName        string      // human-readable name of bot
	botFullName    string      // human-readble full name of the bot
	botID          string      // slack internal bot ID
	users          []testUser  // configured users
	heard          chan string // when the user speaks
	bot.Handler                // bot API for connectors
	sync.RWMutex               // shared mutex for locking connector data structures
}

func (tc *testConnector) Run(stop chan struct{}) {
	tc.Lock()
	// This should never happen, just a bit of defensive coding
	if tc.running {
		tc.Unlock()
		return
	}
	tc.running = true
	tc.Unlock()

	// listen loop
	go func(tc *testConnector) {
		for {
			reader := bufio.NewReader(os.Stdin)

			input, _ := reader.ReadString('\n')
			input = strings.Replace(input, "\n", "", -1)
			input = strings.Replace(input, "\r", "", -1) // should be harmless for Unix
			tc.heard <- input
		}
	}(tc)

loop:
	for {

		select {
		case <-stop:
			tc.Log(bot.Debug, "Received stop in connector")
			break loop
		case input := <-tc.heard:
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
							tc.Log(bot.Info, fmt.Sprintf("Changed current channel to: %s", newchan))
						} else {
							tc.Log(bot.Fatal, "Invalid channel.")
						}
					}
					tc.Unlock()
				case 'U', 'u':
					exists := false
					newuser := input[2:]
					tc.Lock()
					if newuser == "" {
						tc.Log(bot.Fatal, "Invalid 0-length user")
					} else {
						for _, u := range tc.users {
							if u.Name == newuser {
								exists = true
							}
						}
						if exists {
							tc.currentUser = newuser
							tc.Log(bot.Info, fmt.Sprintf("Changed current user to: %s", newuser))
						} else {
							tc.Log(bot.Fatal, "Invalid user.")
						}
					}
					tc.Unlock()
				default:
					tc.Log(bot.Fatal, "Invalid connector command")
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
func (tc *testConnector) sendMessage(ch, msg string) (ret bot.RetVal) {
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
		tc.Log(bot.Error, "Channel not found:", ch)
		return bot.ChannelNotFound
	}
	fmt.Printf("%s: %s\n", ch, msg)
	return bot.Ok
}
