package bot

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/readline"
)

func init() {
	RegisterConnector("terminal", Initialize)
}

const termBotID = "u0000"
const threadIDMax = 65536
const terminalConnectorHelpLine = "Terminal connector: Type '|c?' to list channels, '|u?' to list users, '|t?' for thread help\n"

// Global persistent map of user name to user index
var userIDMap = make(map[string]int)
var userMap = make(map[string]int)

type termUser struct {
	Name                                        string // username / handle
	InternalID                                  string // connector internal identifier
	Email, FullName, FirstName, LastName, Phone string
}

type termconfig struct {
	StartChannel     string // the initial channel
	StartUser        string // the initial userid
	EOF              string // command to send on EOF (ctrl-D), default ";quit"
	Abort            string // command to send on ctrl-c
	BotName          string // the bot's name, required for the robot to hear it's own messages
	Users            []termUser
	Channels         []string
	GenerateNewlines bool // whether to replace the \n sequence with an actual newline
}

// termConnector holds all the relevant data about a connection
type termConnector struct {
	currentChannel   string             // The current channel for the user
	currentUser      string             // The current userid
	currentThread    string             // Active threadID if typingInThread is true
	lastThread       string             // last thread heard from the robot, used with join
	threadCounter    int                // Incrementing integer for assigning thread IDs
	typingInThread   bool               // Tracks whether input is coming from a thread
	generateNewlines bool               // see above
	botName          string             // see above
	eof              string             // command to send on ctrl-d (EOF)
	abort            string             // command to send on ctrl-c (interrupt)
	running          bool               // set on call to Run
	width            int                // width of terminal
	users            []termUser         // configured users
	channels         []string           // the channels the robot is in
	heard            chan string        // when the user speaks
	reader           *readline.Instance // readline for speaking
	robot.Handler                       // bot API for connectors
	sync.RWMutex                        // shared mutex for locking connector data structures
}

var exit = struct {
	kbquit, robotexit bool
	waitchan          chan struct{}
	sync.Mutex
}{
	false, false,
	make(chan struct{}),
	sync.Mutex{},
}

var quitTimeout = 4 * time.Second

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

	var c termconfig

	err := handler.GetProtocolConfig(&c)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to retrieve protocol configuration: %v", err)
	}
	eof := ";quit"
	abort := ";abort"
	if len(c.EOF) > 0 {
		eof = c.EOF
	}
	if len(c.Abort) > 0 {
		abort = c.Abort
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
	if _, ok := userIDMap[termBotID]; !ok {
		firstRunes := []rune(c.BotName)
		firstRunes[0] = unicode.ToUpper(firstRunes[0])
		botUser := termUser{
			Name:       c.BotName,
			InternalID: termBotID,
			Email:      c.BotName + "@example.com",
			FullName:   string(firstRunes) + " Gopherbot",
			FirstName:  string(firstRunes),
			LastName:   "Gopherbot",
			Phone:      "(555)765-0000",
		}
		c.Users = append(c.Users, botUser)
		idx := len(c.Users) - 1
		userMap[c.BotName] = idx
		userIDMap[termBotID] = idx
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
		InterruptPrompt:   "abort",
		EOFPrompt:         "exit",
	})
	if err != nil {
		panic(err)
	}

	tc := &termConnector{
		currentChannel:   c.StartChannel,
		currentUser:      c.StartUser,
		generateNewlines: c.GenerateNewlines,
		botName:          c.BotName,
		eof:              eof,
		abort:            abort,
		channels:         c.Channels,
		running:          false,
		width:            readline.GetScreenWidth(),
		users:            c.Users,
		heard:            make(chan string),
		reader:           rl,
	}

	tc.Handler = handler
	tc.SetTerminalWriter(tc.reader)
	return robot.Connector(tc)
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
	defer func() {
	}()

	// listen loop
	go func(tc *termConnector) {
	readloop:
		for {
			line, err := tc.reader.Readline()
			exit.Lock()
			robotexit := exit.robotexit
			if robotexit {
				exit.Unlock()
				tc.heard <- ""
				break readloop
			}
			kbquit := false
			if err == io.EOF {
				tc.heard <- tc.eof
				kbquit = true
			} else if err == readline.ErrInterrupt {
				tc.heard <- tc.abort
				kbquit = true
			} else if err == nil {
				line = strings.TrimSpace(line)
				if len(line) == 0 {
					tc.reader.Write([]byte(terminalConnectorHelpLine))
				} else {
					if line == "help" {
						tc.reader.Write([]byte(terminalConnectorHelpLine))
					}
					tc.heard <- line
					if line == tc.eof || line == tc.abort {
						kbquit = true
					}
				}
			}
			if kbquit {
				exit.kbquit = true
				exit.Unlock()
				select {
				case <-exit.waitchan:
					break readloop
				case <-time.After(quitTimeout):
					exit.Lock()
					exit.kbquit = false
					exit.Unlock()
					tc.reader.Write([]byte("(timed out waiting for robot to exit; check terminal connector settings 'EOF' and 'Abort')\n"))
				}
			} else {
				exit.Unlock()
			}
		}
	}(tc)

	tc.reader.Write([]byte("Terminal connector running; Type '|c?' to list channels, '|u?' to list users\n"))

	kbquit := false

loop:
	// Main loop and prompting
	for {
		select {
		case <-stop:
			tc.Log(robot.Info, "Received stop in connector")
			exit.Lock()
			kbquit = exit.kbquit
			exit.robotexit = true
			exit.Unlock()
			if kbquit {
				exit.waitchan <- struct{}{}
			} else {
				tc.reader.Write([]byte("Exiting (press <enter> ...)\n"))
			}
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
					newchan = strings.TrimLeft(newchan, " ")
					if newchan == "?" {
						tc.reader.Write([]byte("Available channels:\n"))
						tc.reader.Write([]byte("(direct message); type: '|c'\n"))
						for _, channel := range tc.channels {
							tc.reader.Write([]byte(fmt.Sprintf("'%s'; type: '|c%s'\n", channel, channel)))
						}
						continue
					}
					tc.Lock()
					if newchan == "" {
						tc.currentChannel = ""
						tc.reader.SetPrompt(fmt.Sprintf("c:(direct)/u:%s -> ", tc.currentUser))
						tc.reader.Write([]byte("Changed current channel to: direct message\n"))
						tc.typingInThread = false
					} else {
						for _, ch := range tc.channels {
							if ch == newchan {
								exists = true
								break
							}
						}
						if exists {
							tc.currentChannel = newchan
							tc.typingInThread = false
							tc.reader.SetPrompt(fmt.Sprintf("c:%s/u:%s -> ", tc.currentChannel, tc.currentUser))
							tc.reader.Write([]byte(fmt.Sprintf("Changed current channel to: %s\n", newchan)))
						} else {
							tc.reader.Write([]byte("Invalid channel\n"))
						}
					}
					tc.Unlock()
				case 'J', 'j':
					tc.RLock()
					lastThread := tc.lastThread
					tc.RUnlock()
					if len(lastThread) == 0 {
						tc.reader.Write([]byte("(sorry, I don't see a thread to join)\n"))
						continue
					}
					tc.Lock()
					tc.typingInThread = true
					tc.currentThread = lastThread
					tc.reader.SetPrompt(fmt.Sprintf("c:%s(%s)/u:%s -> ", tc.currentChannel, tc.currentThread, tc.currentUser))
					tc.reader.Write([]byte(fmt.Sprintf("(now typing in thread: %s)\n", tc.currentThread)))
					tc.Unlock()
				case 'T', 't':
					setThread := input[2:]
					setThread = strings.TrimLeft(setThread, " ")
					if setThread == "?" {
						tc.reader.Write([]byte("Use '|t' to toggle typing in a thread, '|t<string>' to set the current thread ID, or '|j' to join the robot's thread\n"))
						continue
					}
					tc.Lock()
					if len(setThread) == 0 {
						tc.typingInThread = !tc.typingInThread
						if tc.typingInThread {
							tc.currentThread = fmt.Sprintf("%04x", tc.threadCounter%threadIDMax)
						}
					} else {
						tc.typingInThread = true
						tc.currentThread = setThread
					}
					if tc.typingInThread {
						tc.reader.SetPrompt(fmt.Sprintf("c:%s(%s)/u:%s -> ", tc.currentChannel, tc.currentThread, tc.currentUser))
						tc.reader.Write([]byte(fmt.Sprintf("(now typing in thread: %s)\n", tc.currentThread)))
					} else {
						tc.reader.SetPrompt(fmt.Sprintf("c:%s/u:%s -> ", tc.currentChannel, tc.currentUser))
						tc.reader.Write([]byte("(typing in channel now)\n"))
					}
					tc.Unlock()
				case 'U', 'u':
					exists := false
					newuser := input[2:]
					newuser = strings.TrimLeft(newuser, " ")
					if newuser == "?" {
						tc.reader.Write([]byte("Available users:\n"))
						for _, user := range tc.users {
							tc.reader.Write([]byte(fmt.Sprintf("'%s'; type: '|u%s'\n", user.Name, user.Name)))
						}
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
							tc.reader.SetPrompt(fmt.Sprintf("c:%s/u:%s -> ", tc.currentChannel, tc.currentUser))
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
				var threadID, messageID string
				tc.Lock()
				tc.threadCounter++
				messageNumber := tc.threadCounter
				tc.Unlock()
				if tc.typingInThread {
					messageID = fmt.Sprintf("%04x", messageNumber%threadIDMax)
					threadID = tc.currentThread
				} else {
					threadID = fmt.Sprintf("%04x", messageNumber%threadIDMax)
					messageID = threadID
				}
				if tc.generateNewlines {
					input = strings.ReplaceAll(input, `\n`, "\n")
				}
				botMsg := &robot.ConnectorMessage{
					Protocol:        "terminal",
					UserName:        tc.currentUser,
					UserID:          ui.InternalID,
					ChannelName:     tc.currentChannel,
					ChannelID:       channelID,
					MessageID:       messageID,
					ThreadedMessage: tc.typingInThread,
					ThreadID:        threadID,
					MessageText:     input,
					DirectMessage:   direct,
				}
				tc.RLock()
				tc.IncomingMessage(botMsg)
				tc.RUnlock()
			}
		}
	}
	if !kbquit {
		<-tc.heard
	}
	tc.reader.Write([]byte("Terminal connector finished\n"))
	tc.reader.Close()
}

func (tc *termConnector) MessageHeard(u, c string) {
	return
}

func (tc *termConnector) getUserInfo(u string) (*termUser, bool) {
	var i int
	var exists bool
	if id, ok := tc.ExtractID(u); ok {
		i, exists = userIDMap[id]
	} else {
		i, exists = userMap[u]
	}
	if exists {
		return &tc.users[i], true
	}
	return nil, false
}

func (tc *termConnector) getChannel(c string) string {
	if ch, ok := tc.ExtractID(c); ok {
		return strings.TrimPrefix(ch, "#")
	}
	return c
}

func (tc *termConnector) checkSendSelf(ch, thr, msg string, f robot.MessageFormat) {
	var threadID, messageID string
	var threadedMessage bool
	tc.Lock()
	tc.threadCounter++
	messageNumber := tc.threadCounter
	tc.Unlock()
	if len(thr) > 0 {
		threadedMessage = true
		messageID = fmt.Sprintf("%04x", messageNumber%threadIDMax)
		threadID = thr
	} else {
		threadID = fmt.Sprintf("%04x", messageNumber%threadIDMax)
		messageID = threadID
	}
	tc.Log(robot.Debug, "forwarding message id '%s' from the robot %s/%s", messageID, tc.botName, termBotID)
	botMsg := &robot.ConnectorMessage{
		Protocol:        "terminal",
		UserName:        tc.botName,
		UserID:          termBotID,
		ChannelName:     ch,
		ChannelID:       "#" + ch,
		MessageID:       messageID,
		ThreadedMessage: threadedMessage,
		SelfMessage:     true,
		ThreadID:        threadID,
		MessageText:     msg,
	}
	tc.RLock()
	tc.IncomingMessage(botMsg)
	tc.RUnlock()
}

// SetUserMap lets Gopherbot provide a mapping of usernames to user IDs
func (tc *termConnector) SetUserMap(map[string]string) {
	return
}

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (tc *termConnector) GetProtocolUserAttribute(u, attr string) (value string, ret robot.RetVal) {
	var user *termUser
	var exists bool
	if user, exists = tc.getUserInfo(u); !exists {
		return "", robot.UserNotFound
	}
	switch attr {
	case "email":
		return user.Email, robot.Ok
	case "internalid":
		return user.InternalID, robot.Ok
	case "realname", "fullname", "real name", "full name":
		return user.FullName, robot.Ok
	case "firstname", "first name":
		return user.FirstName, robot.Ok
	case "lastname", "last name":
		return user.LastName, robot.Ok
	case "phone":
		return user.Phone, robot.Ok
	// that's all the attributes we can currently get from slack
	default:
		return "", robot.AttributeNotFound
	}
}

// SendProtocolChannelThreadMessage sends a message to a channel
func (tc *termConnector) SendProtocolChannelThreadMessage(ch, thr, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	channel := tc.getChannel(ch)
	return tc.sendMessage(channel, thr, msg, f)
}

// SendProtocolChannelMessage sends a message to a channel
func (tc *termConnector) SendProtocolUserChannelThreadMessage(uid, uname, ch, thr, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	channel := tc.getChannel(ch)
	msg = "@" + uname + " " + msg
	return tc.sendMessage(channel, thr, msg, f)
}

// SendProtocolUserMessage sends a direct message to a user
func (tc *termConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	var user *termUser
	var exists bool
	if user, exists = tc.getUserInfo(u); !exists {
		return robot.UserNotFound
	}
	return tc.sendMessage(fmt.Sprintf("(dm:%s)", user.Name), "", msg, f)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
// Only useful for connectors that require it, a noop otherwise
func (tc *termConnector) JoinChannel(c string) (ret robot.RetVal) {
	return robot.Ok
}
