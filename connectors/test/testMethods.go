package test

import (
	"errors"
	"fmt"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

/* testMethods.go - methods specific to the test connector */

// SendBotMessage for tests to send messages to the 'bot
func (tc *TestConnector) SendBotMessage(msg *TestMessage) {
	tc.RLock()
	if msg.Channel != "" {
		exists := false
		for _, ch := range tc.channels {
			if ch == msg.Channel {
				exists = true
				break
			}
		}
		if !exists {
			tc.reportError("Invalid channel: %s", msg.Channel)
		}
	}
	if msg.User == "" {
		tc.reportError("Invalid 0-length user")
	} else {
		exists := false
		for _, u := range tc.users {
			if u.InternalID == msg.User {
				exists = true
			}
		}
		if !exists {
			tc.reportError("Invalid user: %s", msg.User)
		}
	}
	tc.RUnlock()
	select {
	case tc.listener <- msg:
		tc.reportLog("Message sent to robot: %#v", msg)
	case <-time.After(200 * time.Millisecond):
		tc.reportError("Timed out sending; user: %q, channel: %q, message: %q", msg.User, msg.Channel, msg.Message)
	}
}

// GetBotMessage for tests to get replies
func (tc *TestConnector) GetBotMessage() (*TestMessage, error) {
	select {
	case incoming := <-tc.speaking:
		message := incoming.Message
		if len(incoming.Message) > 16 {
			message = incoming.Message[0:16] + " ..."
		}
		tc.reportLog("Reply received from robot: u:%s, c:%s, m:%s, t:%t", incoming.User, incoming.Channel, message, incoming.Threaded)
		time.Sleep(100 * time.Millisecond)
		return incoming, nil
	case <-time.After(4 * time.Second):
		return nil, errors.New("timeout waiting for reply from robot")
	}
}

func (tc *TestConnector) DrainBotMessages() []*TestMessage {
	messages := make([]*TestMessage, 0)
	for {
		select {
		case msg := <-tc.speaking:
			messages = append(messages, msg)
		default:
			return messages
		}
	}
}

func (tc *TestConnector) reportError(format string, args ...interface{}) {
	tc.RLock()
	reporter := tc.test
	tc.RUnlock()
	if reporter != nil {
		reporter.Errorf(format, args...)
		return
	}
	if tc.Handler != nil {
		tc.Log(robot.Error, fmt.Sprintf(format, args...))
	}
}

func (tc *TestConnector) reportLog(format string, args ...interface{}) {
	tc.RLock()
	reporter := tc.test
	tc.RUnlock()
	if reporter != nil {
		reporter.Logf(format, args...)
		return
	}
	if tc.Handler != nil {
		tc.Log(robot.Debug, fmt.Sprintf(format, args...))
	}
}
