// Package test implements a test connector for automated testing.

package test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// TestMessage is for sending messages to the robot
type TestMessage struct {
	User, Channel, Message string
}

// TestConnector holds all the relevant data about a connection
type TestConnector struct {
	botName       string            // human-readable name of bot
	botFullName   string            // human-readble full name of the bot
	botID         string            // slack internal bot ID
	users         []testUser        // configured users
	channels      []string          // the channels the robot is in
	listener      chan *TestMessage // input channel for test functions to send messages from a user
	speaking      chan *TestMessage // output channel for test functions to get messages from the bot
	test          *testing.T        // for the connector to log
	robot.Handler                   // bot API for connectors
	sync.RWMutex                    // shared mutex for locking connector data structures
}

// Run starts the main loop for the test connector
func (tc *TestConnector) Run(stop <-chan struct{}) {

loop:
	for {
		select {
		case <-stop:
			tc.Log(robot.Debug, "Received stop in connector")
			tc.test.Log("Received stop in connector")
			break loop
		case msg := <-tc.listener:
			var userName, channelID string
			i, exists := userIDMap[msg.User]
			if exists {
				userName = tc.users[i].Name
			}
			direct := false
			if len(msg.Channel) > 0 {
				channelID = "#" + msg.Channel
			} else {
				direct = true
			}
			botMsg := &robot.ConnectorMessage{
				Protocol:      "Test",
				UserName:      userName,
				UserID:        msg.User,
				ChannelName:   msg.Channel,
				ChannelID:     channelID,
				DirectMessage: direct,
				MessageText:   msg.Message,
				MessageObject: msg,
				Client:        tc,
			}
			tc.IncomingMessage(botMsg)
		}
	}
}

// Public 'bot methods all call sendMessage to send a message to a user/channel
func (tc *TestConnector) sendMessage(msg *BotMessage) (ret robot.RetVal) {
	if msg.Channel == "" && msg.User == "" {
		tc.test.Errorf("Invalid empty user and channel")
		return robot.ChannelNotFound
	}
	if msg.Channel != "" { // direct message
		found := false
		tc.RLock()
		for _, channel := range tc.channels {
			if channel == msg.Channel {
				found = true
				break
			}
		}
		tc.RUnlock()
		if !found {
			tc.test.Errorf("Channel not found: %s", msg.Channel)
			return robot.ChannelNotFound
		}
	}
	if msg.User != "" { // speaking in channel, not talking to user
		found := false
		tc.RLock()
		for _, user := range tc.users {
			if user.Name == msg.User {
				found = true
				break
			}
		}
		tc.RUnlock()
		if !found {
			tc.test.Errorf("User not found: %s", msg.User)
			return robot.UserNotFound
		}
	}
	spoken := &TestMessage{
		User:    msg.User,
		Channel: msg.Channel,
	}
	switch msg.Format {
	case robot.Fixed:
		spoken.Message = strings.ToUpper(msg.Message)
	case robot.Variable:
		spoken.Message = strings.ToLower(msg.Message)
	case robot.Raw:
		spoken.Message = msg.Message
	}
	select {
	case tc.speaking <- spoken:
	case <-time.After(200 * time.Millisecond):
		return robot.TimeoutExpired
	}

	return robot.Ok
}
