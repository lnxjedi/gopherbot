package test

import (
	"testing"
	"time"
)

/* testMethods.go - methods specific to the test connector */

func (tc *TestConnector) SetTest(t *testing.T) {
	tc.Lock()
	tc.test = t
	tc.Unlock()
}

// SendBotMessage, for tests to send messages to the 'bot
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
			tc.test.Failf("Invalid channel: %s", msg.Channel)
		}
	}
	if msg.User == "" {
		tc.test.Failf("Invalid 0-length user")
	} else {
		exists := false
		for _, u := range tc.users {
			if u.Name == msg.User {
				exists = true
			}
		}
		if !exists {
			tc.test.Failf("Invalid user: %s", msg.User)
		}
	}
	tc.RUnlock()
	select {
	case tc.Listening <- msg:
	case time.After(200 * time.Millisecond):
		tc.test.Failf("Timed out sending; user: \"%s\", channel: \"%s\", message: \"%s\"", msg.User, msg.Channel, msg.Message)
	}
}

// GetBotMessage, for tests to get replies
func (tc *TestConnector) GetBotMessage() *TestMessage {
	select {
	case incoming := <-tc.Speaking:
		return incoming
	case time.After(2 * time.Second):
		tc.test.Failf("Timed out waiting for reply in GetBotMessage")
		return nil
	}
}
