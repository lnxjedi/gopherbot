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

	"github.com/lnxjedi/readline"
	"github.com/lnxjedi/robot"
)

func init() {
	RegisterPreload("connectors/terminal.so")
	RegisterConnector("terminal", Initialize)
}

// Global persistent map of user name to user index
var userIDMap = make(map[string]int)
var userMap = make(map[string]int)

type termUser struct {
	Name                                        string // username / handle
	InternalID                                  string // connector internal identifier
	Email, FullName, FirstName, LastName, Phone string
}

type termconfig struct {
	StartChannel string // the initial channel
	StartUser    string // the initial userid
	EOF          string // command to send on EOF (ctrl-D), default ";quit"
	Abort        string // command to send on ctrl-c
	Users        []termUser
	Channels     []string
}

// termConnector holds all the relevant data about a connection
type termConnector struct {
	currentChannel string             // The current channel for the user
	currentUser    string             // The current userid
	eof            string             // command to send on ctrl-d (EOF)
	abort          string             // command to send on ctrl-c (interrupt)
	running        bool               // set on call to Run
	width          int                // width of terminal
	users          []termUser         // configured users
	channels       []string           // the channels the robot is in
	heard          chan string        // when the user speaks
	reader         *readline.Instance // readline for speaking
	robot.Handler                     // bot API for connectors
	sync.RWMutex                      // shared mutex for locking connector data structures
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
		currentChannel: c.StartChannel,
		currentUser:    c.StartUser,
		eof:            eof,
		abort:          abort,
		channels:       c.Channels,
		running:        false,
		width:          readline.GetScreenWidth(),
		users:          c.Users,
		heard:          make(chan string),
		reader:         rl,
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
				tc.heard <- line
				line = strings.TrimSpace(line)
				if line == tc.eof || line == tc.abort {
					kbquit = true
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

	tc.reader.Write([]byte("Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user\n"))

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
				botMsg := &robot.ConnectorMessage{
					Protocol:      "terminal",
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

// SendProtocolChannelMessage sends a message to a channel
func (tc *termConnector) SendProtocolChannelMessage(ch string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	channel := tc.getChannel(ch)
	return tc.sendMessage(channel, msg, f)
}

// SendProtocolChannelMessage sends a message to a channel
func (tc *termConnector) SendProtocolUserChannelMessage(uid, uname, ch, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	channel := tc.getChannel(ch)
	msg = "@" + uname + " " + msg
	return tc.sendMessage(channel, msg, f)
}

// SendProtocolUserMessage sends a direct message to a user
func (tc *termConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	var user *termUser
	var exists bool
	if user, exists = tc.getUserInfo(u); !exists {
		return robot.UserNotFound
	}
	return tc.sendMessage(fmt.Sprintf("(dm:%s)", user.Name), msg, f)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
// Only useful for connectors that require it, a noop otherwise
func (tc *termConnector) JoinChannel(c string) (ret robot.RetVal) {
	return robot.Ok
}
