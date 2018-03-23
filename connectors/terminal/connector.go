// Package terminal implements a terminal console connector for plugin development
// and bot testing; eventually a test framework will be built around it.
package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/bot"
)

// termConnector holds all the relevant data about a connection
type termConnector struct {
	currentChannel string        // The current channel for the user
	currentUser    string        // The current userid
	channels       []string      // the channels the robot is in
	running        bool          // set on call to Run
	botName        string        // human-readable name of bot
	botFullName    string        // human-readble full name of the bot
	botID          string        // slack internal bot ID
	users          []termUser    // configured users
	heard          chan string   // when the user speaks
	speaking       chan struct{} // when the bot speaks
	bot.Handler                  // bot API for connectors
	sync.RWMutex                 // shared mutex for locking connector data structures
}

// Message send delay; slack has problems with scrolling if messages fly out
// too fast.
const typingDelay = 200 * time.Millisecond
const msgDelay = 1 * time.Second

// Bursting constants; we allow the robot to send a maximum of `burstMessages`
// in a `burstWindow` window; above the burst limit we slow messages down to
// 1 / sec.
const burstMessages = 14            // maximum burst
const burstWindow = 4 * time.Second // window in which to allow the burst
const coolDown = 21 * time.Second   // cooldown time after bursting

func (tc *termConnector) Run(stop chan struct{}) {
	tc.Lock()
	// This should never happen, just a bit of defensive coding
	if tc.running {
		tc.Unlock()
		return
	}
	tc.running = true
	tc.Unlock()

	// send loop
	go func(tc *termConnector) {
		// See bursting constants above.
		var burstTime time.Time
		mtimes := make([]time.Time, burstMessages)
		current := 0 // index of the current message send time
		for {
			send := <-messages
			msgTime := time.Now()
			mtimes[current] = msgTime
			windowStartMsg := current + 1
			if windowStartMsg == (burstMessages - 1) {
				windowStartMsg = 0
			}
			current += 1
			if current == (burstMessages - 1) {
				current = 0
			}
			tc.speaking <- struct{}{}
			time.Sleep(typingDelay)
			fmt.Printf("%s: %s\n", send.channel, send.message)
			timeSinceBurst := msgTime.Sub(burstTime)
			if msgTime.Sub(mtimes[windowStartMsg]) < burstWindow || timeSinceBurst < coolDown {
				if timeSinceBurst > coolDown {
					burstTime = msgTime
				}
				tc.Log(bot.Debug, fmt.Sprintf("Burst limit exceeded, delaying next message by %v", msgDelay))
				// if we've sent `burstMessages` messages in less than the `burstWindow`
				// window, delay the next message by `msgDelay`.
				time.Sleep(msgDelay)
			}
		}
	}(tc)

	// listen loop
	go func(tc *termConnector) {
		for {
			reader := bufio.NewReader(os.Stdin)

			input, _ := reader.ReadString('\n')
			input = strings.Replace(input, "\n", "", -1)
			input = strings.Replace(input, "\r", "", -1) // should be harmless for Unix
			tc.heard <- input
		}
	}(tc)

	fmt.Println("Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user")

	var lastSpoke time.Time
	prompted := false
	speaking := false

loop:
	// Main loop and prompting
	for {
		if !speaking && !prompted {
			tc.RLock()
			fmt.Printf("c:%s/u:%s -> ", tc.currentChannel, tc.currentUser)
			tc.RUnlock()
			prompted = true
		}

		select {
		case <-stop:
			tc.Log(bot.Debug, "Received stop in connector")
			break loop
		case input := <-tc.heard:
			prompted = false
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
						fmt.Println("Changed current channel to: direct message")
					} else {
						for _, ch := range tc.channels {
							if ch == newchan {
								exists = true
								break
							}
						}
						if exists {
							fmt.Printf("Changed current channel to: %s\n", newchan)
							tc.currentChannel = newchan
						} else {
							fmt.Println("Invalid channel.")
						}
					}
					tc.Unlock()
				case 'U', 'u':
					exists := false
					newuser := input[2:]
					tc.Lock()
					if newuser == "" {
						fmt.Println("Invalid 0-length user")
					} else {
						for _, u := range tc.users {
							if u.Name == newuser {
								exists = true
							}
						}
						if exists {
							tc.currentUser = newuser
							fmt.Printf("Changed current user to: %s\n", newuser)
						} else {
							fmt.Println("Invalid user.")
						}
					}
					tc.Unlock()
				default:
					fmt.Println("Invalid terminal connector command")
				}
			} else {
				tc.RLock()
				tc.IncomingMessage(tc.currentChannel, tc.currentUser, input)
				tc.RUnlock()
			}
		case <-tc.speaking:
			if !speaking {
				fmt.Println()
			}
			speaking = true
			prompted = false
			lastSpoke = time.Now()
		case <-time.After(100 * time.Millisecond):
			if time.Now().Sub(lastSpoke) > time.Second {
				speaking = false
			}
		}
	}
}

type sendMessage struct {
	message, channel string
}

var messages = make(chan *sendMessage)

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
	messages <- &sendMessage{
		message: msg,
		channel: ch,
	}
	return bot.Ok
}
