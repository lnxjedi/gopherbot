// Package terminal implements a terminal console connector for plugin development
// and bot testing; eventually a test framework will be built around it.
package terminal

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/chzyer/readline"
	"github.com/lnxjedi/gopherbot/robot"
)

// termConnector holds all the relevant data about a connection
type termConnector struct {
	currentChannel string             // The current channel for the user
	currentUser    string             // The current userid
	eof            string             // command to send on ctrl-d (EOF)
	running        bool               // set on call to Run
	users          []termUser         // configured users
	channels       []string           // the channels the robot is in
	heard          chan string        // when the user speaks
	reader         *readline.Instance // readline for speaking
	robot.Handler                     // bot API for connectors
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
			line, err := tc.reader.Readline()
			if err == io.EOF {
				tc.heard <- tc.eof
			} else {
				tc.heard <- line
			}
		}
	}(tc)
	tc.reader.Write([]byte("Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user\n"))

loop:
	// Main loop and prompting
	for {

		select {
		case <-stop:
			tc.Log(robot.Debug, "Received stop in connector")
			fmt.Println("Exiting (press enter)")
			break loop
		case input := <-tc.heard:
			if len(input) == 0 {
				evs := tc.GetEventStrings()
				if len(*evs) > 0 {
					tc.reader.Write([]byte(fmt.Sprintf("Events gathered: %s\n", strings.Join(*evs, ", "))))
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
					if newchan == "?" {
						chanlist := []string{"Available channels; '|c' for direct message:"}
						for _, channel := range tc.channels {
							chanlist = append(chanlist, fmt.Sprintf("%s ('|c%s')", channel, channel))
						}
						tc.reader.Write([]byte(strings.Join(chanlist, "\n")))
						tc.reader.Write([]byte("\n"))
						continue
					}
					tc.Lock()
					if newchan == "" {
						tc.currentChannel = ""
						tc.reader.SetPrompt(fmt.Sprintf("c:(direct)/u:%s -> ", tc.currentUser))
						tc.reader.Write([]byte("Changed current channel to: direct message\n"))
					} else {
						for _, ch := range tc.channels {
							if ch == newchan {
								exists = true
								break
							}
						}
						if exists {
							tc.currentChannel = newchan
							tc.reader.SetPrompt(fmt.Sprintf("c:%s/u:%s -> ", tc.currentChannel, tc.currentUser))
							tc.reader.Write([]byte(fmt.Sprintf("Changed current channel to: %s\n", newchan)))
						} else {
							tc.reader.Write([]byte("Invalid channel\n"))
						}
					}
					tc.Unlock()
				case 'U', 'u':
					exists := false
					newuser := input[2:]
					if newuser == "?" {
						userlist := []string{"Available users:"}
						for _, user := range tc.users {
							userlist = append(userlist, fmt.Sprintf("%s ('|u%s')", user.Name, user.Name))
						}
						tc.reader.Write([]byte(strings.Join(userlist, "\n")))
						tc.reader.Write([]byte("\n"))
						continue
					}
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
							tc.reader.SetPrompt(fmt.Sprintf("c:%s/u:%s -> ", tc.currentUser, tc.currentChannel))
							tc.reader.Write([]byte(fmt.Sprintf("Changed current user to: %s\n", newuser)))
						} else {
							tc.reader.Write([]byte("Invalid user\n"))
						}
					}
					tc.Unlock()
				default:
					tc.reader.Write([]byte("Invalid terminal connector command\n"))
				}
			} else {
				var channelID string
				direct := false
				if len(tc.currentChannel) > 0 {
					channelID = "#" + tc.currentChannel
				} else {
					direct = true
				}
				i := userMap[tc.currentUser]
				ui := tc.users[i]
				botMsg := &robot.ConnectorMessage{
					Protocol:      "Terminal",
					UserName:      tc.currentUser,
					UserID:        ui.InternalID,
					ChannelName:   tc.currentChannel,
					ChannelID:     channelID,
					MessageText:   input,
					DirectMessage: direct,
				}
				tc.RLock()
				tc.IncomingMessage(botMsg)
				tc.RUnlock()
			}
		}
	}
}
