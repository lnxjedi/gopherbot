// Package terminal implements a terminal console connector for plugin development
// and bot testing; eventually a test framework will be built around it.
package terminal

import (
	"fmt"
	"strings"
	"sync"

	"github.com/chzyer/readline"
	"github.com/lnxjedi/gopherbot/bot"
)

// termConnector holds all the relevant data about a connection
type termConnector struct {
	currentChannel string             // The current channel for the user
	currentUser    string             // The current userid
	running        bool               // set on call to Run
	botName        string             // human-readable name of bot
	botFullName    string             // human-readble full name of the bot
	botID          string             // slack internal bot ID
	users          []termUser         // configured users
	channels       []string           // the channels the robot is in
	heard          chan string        // when the user speaks
	reader         *readline.Instance // readline for speaking
	bot.Handler                       // bot API for connectors
	sync.RWMutex                      // shared mutex for locking connector data structures
}

func (tc *termConnector) Run(stop <-chan struct{}) {
	tc.Lock()
	// This should never happen, just a bit of defensive coding
	if tc.running {
		tc.Unlock()
		return
	}
	tc.running = true
	tc.Unlock()
	defer tc.reader.Close()

	// listen loop
	go func(tc *termConnector) {
		for {
			line, _ := tc.reader.Readline()
			tc.heard <- line
		}
	}(tc)

	tc.reader.Write([]byte("Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user\n"))

loop:
	// Main loop and prompting
	for {

		select {
		case <-stop:
			tc.Log(bot.Debug, "Received stop in connector")
			fmt.Println("Exiting (press enter)")
			break loop
		case input := <-tc.heard:
			if len(input) == 0 {
				ev := bot.GetEvents()
				if len(*ev) > 0 {
					evs := make([]string, len(*ev))
					for i, e := range *ev {
						evs[i] = e.String()
					}
					tc.reader.Write([]byte(fmt.Sprintf("Events gathered: %s\n", strings.Join(evs, ", "))))
				}
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
						tc.reader.Write([]byte("Changed current channel to: direct message\n"))
						tc.reader.SetPrompt(fmt.Sprintf("c:(direct)/u:%s -> ", tc.currentUser))
					} else {
						for _, ch := range tc.channels {
							if ch == newchan {
								exists = true
								break
							}
						}
						if exists {
							tc.reader.Write([]byte(fmt.Sprintf("Changed current channel to: %s\n", newchan)))
							tc.currentChannel = newchan
							tc.reader.SetPrompt(fmt.Sprintf("c:%s/u:%s -> ", tc.currentChannel, tc.currentUser))
						} else {
							tc.reader.Write([]byte("Invalid channel\n"))
						}
					}
					tc.Unlock()
				case 'U', 'u':
					exists := false
					newuser := input[2:]
					tc.Lock()
					if newuser == "" {
						tc.reader.Write([]byte("Invalid 0-length user\n"))
					} else {
						for _, u := range tc.users {
							if u.Name == newuser {
								exists = true
							}
						}
						if exists {
							tc.currentUser = newuser
							tc.reader.Write([]byte(fmt.Sprintf("Changed current user to: %s\n", newuser)))
							tc.reader.SetPrompt(fmt.Sprintf("c:%s/u:%s -> ", tc.currentUser, tc.currentChannel))
						} else {
							tc.reader.Write([]byte("Invalid user\n"))
						}
					}
					tc.Unlock()
				default:
					tc.reader.Write([]byte("Invalid terminal connector command\n"))
				}
			} else {
				tc.RLock()
				tc.IncomingMessage(tc.currentChannel, tc.currentUser, input, "terminal", bot.Terminal, struct{}{})
				tc.RUnlock()
			}
		}
	}
}

func (tc *termConnector) sendMessage(ch, msg string) (ret bot.RetVal) {
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
	tc.reader.Write([]byte(fmt.Sprintf("%s: %s\n", ch, msg)))
	return bot.Ok
}
