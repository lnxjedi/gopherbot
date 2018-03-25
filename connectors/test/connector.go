// Package test implements a test connector for automated testing.

package test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/bot"
)

// TestMessage is for sending/receiving messages
type TestMessage struct {
	User, Channel, Message string
}

// TestConnector holds all the relevant data about a connection
type TestConnector struct {
	channels     []string            // the channels the robot is in
	botName      string              // human-readable name of bot
	botFullName  string              // human-readble full name of the bot
	botID        string              // slack internal bot ID
	users        []testUser          // configured users
	listener     <-chan *TestMessage // input channel for test functions to send messages from a user
	speaking     chan *TestMessage   // output channel for test functions to get messages from the bot
	test         *testing.T          // for the connector to log
	bot.Handler                      // bot API for connectors
	sync.RWMutex                     // shared mutex for locking connector data structures
}

func (tc *TestConnector) Run(stop <-chan struct{}) {

loop:
	for {

		select {
		case <-stop:
			tc.Test.Log(bot.Debug, "Received stop in connector")
			break loop
		case msg := <-tc.Listener:
			tc.IncomingMessage(msg.Channel, msg.User, msg.Message)
		}
	}
}

// Public 'bot methods all call sendMessage to send a message to a user/channel
func (tc *TestConnector) sendMessage(msg *TestMessage) (ret bot.RetVal) {
	if msg.Channel == "" && msg.User == "" {
		tc.test.Failf("Invalid empty user and channel")
		return bot.ChannelNotFound
	}
	if msg.Channel != "" { // direct message
		found := false
		tc.RLock()
		for _, channel := range tc.channels {
			if channel == ch {
				found = true
				break
			}
		}
		tc.RUnlock()
		if !found {
			tc.test.Failf("Channel not found: %s", ch)
			return bot.ChannelNotFound
		}
	}
	if msg.User != "" { // direct message
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
			tc.test.Failf("Channel not found: %s", ch)
			return bot.ChannelNotFound
		}
	}
	select {
	case tc.Speaking <- msg:
	case time.After(200 * time.Millisecond):
		tc.test.Failf("Timed out waiting for GetBotMessage, user: \"%s\", channel: \"%s\", message: \"%s\"", msg.User, msg.Channel, msg.Message)
		return bot.TimeoutExpired
	}

	return bot.Ok
}
