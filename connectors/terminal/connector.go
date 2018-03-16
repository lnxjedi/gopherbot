// Package terminal implements a terminal console connector for plugin development
// and bot testing; eventually a test framework will be built around it.
package terminal

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/bot"
)

type config struct {
	StartChannel string // the initial channel
	StartUser    string // the initial userid
	BotName      string // the short name used for addressing the robot
	BotFullName  string // the full name of the bot
}

// termConnector holds all the relevant data about a connection
type termConnector struct {
	currentChannel string        // The current channel for the user
	currentUser    string        // The current userid
	channels       []string      // the channels the robot is in
	running        bool          // set on call to Run
	botName        string        // human-readable name of bot
	botFullName    string        // human-readble full name of the bot
	botID          string        // slack internal bot ID
	listening      chan string   // when the user speaks
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

	tc := &termConnector{
		currentChannel: c.StartChannel,
		currentUser:    c.StartUser,
		channels:       make([]string, 0),
		running:        false,
		botName:        c.BotName,
		botFullName:    c.BotFullName,
		botID:          "deadbeef",
		listening:      make(chan string),
		speaking:       make(chan struct{}),
	}

	tc.Handler = robot
	tc.SetFullName(tc.botFullName)
	tc.Log(bot.Debug, "Set bot full name to", tc.botFullName)
	tc.SetName(tc.botName)
	tc.Log(bot.Info, "Set bot name to", tc.botName)

	go tc.startSendLoop()

	return bot.Connector(tc)
}

func (tc *termConnector) Run(stop chan struct{}) {
	tc.Lock()
	// This should never happen, just a bit of defensive coding
	if tc.running {
		tc.Unlock()
		return
	}
	tc.running = true
	tc.Unlock()

	go func(tc *termConnector) {
		for {
			reader := bufio.NewReader(os.Stdin)

			input, _ := reader.ReadString('\n')
			input = strings.Replace(input, "\n", "", -1)
			tc.listening <- input
		}
	}(tc)

	fmt.Println("Terminal connector running; Use '|C<channel>' to change channel, or '|U<user' to change user")

loop:

	for {
		tc.RLock()
		fmt.Printf("%s/%s -> ", tc.currentUser, tc.currentChannel)
		tc.RUnlock()

		select {
		case <-stop:
			tc.Log(bot.Debug, "Received stop in connector")
			break loop
		case input := <-tc.listening:
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
							fmt.Printf("Changed current channel to: %s", newchan)
							tc.currentChannel = newchan
						} else {
							fmt.Println("Invalid channel.")
						}
					}
					tc.Unlock()
				case 'U', 'u':
					newuser := input[2:]
					tc.Lock()
					if newuser == "" {
						fmt.Println("Invalid 0-length user")
					} else {
						tc.currentUser = newuser
						fmt.Printf("Changed current user to: %s", newuser)
					}
					tc.Unlock()
				default:
					fmt.Println("Invalid terminal connector command")
				}
			} else {
				tc.RLock()
				tc.IncomingMessage(tc.currentUser, tc.currentChannel, input)
				tc.RUnlock()
			}
		case <-tc.speaking:
			lastSpoke := time.Now()
			fmt.Println()
			for {
				select {
				case <-stop:
					tc.Log(bot.Debug, "Received stop in connector")
					break loop
				case <-tc.speaking:
					lastSpoke = time.Now()
				case <-time.After(100 * time.Millisecond):
					if time.Now().Sub(lastSpoke) > 2*time.Second {
						break
					}
				}
			}
		}
	}
}

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (tc *termConnector) GetProtocolUserAttribute(u, attr string) (value string, ret bot.RetVal) {
	switch attr {
	case "email":
		return "jdoe@example.com", bot.Ok
	case "internalID":
		return "u12345", bot.Ok
	case "realName", "fullName":
		return "J. Doe", bot.Ok
	case "firstName":
		return "J", bot.Ok
	case "lastName":
		return "Doe", bot.Ok
	case "phone":
		return "(555)765-4321", bot.Ok
	// that's all the attributes we can currently get from slack
	default:
		return "", bot.AttributeNotFound
	}
}

type sendMessage struct {
	message, channel string
}

var messages = make(chan *sendMessage)

func (tc *termConnector) startSendLoop() {
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
}

func (tc *termConnector) sendMessage(ch, msg string) (ret bot.RetVal) {
	found := false
	tc.RLock()
	if ch == "" {
		ch = "(direct message)"
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

// SendProtocolChannelMessage sends a message to a channel
func (tc *termConnector) SendProtocolChannelMessage(ch string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	return tc.sendMessage(ch, msg)
}

// SendProtocolChannelMessage sends a message to a channel
func (tc *termConnector) SendProtocolUserChannelMessage(u, ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	msg = "@" + u + ": " + msg
	return tc.sendMessage(ch, msg)
}

// SendProtocolUserMessage sends a direct message to a user
func (tc *termConnector) SendProtocolUserMessage(u string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	return tc.sendMessage("", msg)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (tc *termConnector) JoinChannel(c string) (ret bot.RetVal) {
	if c == "" {
		return bot.Ok
	}
	found := false
	tc.Lock()
	for _, channel := range tc.channels {
		if channel == c {
			found = true
			break
		}
	}
	if !found {
		tc.channels = append(tc.channels, c)
	}
	tc.Unlock()
	return bot.Ok
}
