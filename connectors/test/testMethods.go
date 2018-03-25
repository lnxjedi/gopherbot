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
			tc.test.Errorf("Invalid channel: %s", msg.Channel)
		}
	}
	if msg.User == "" {
		tc.test.Errorf("Invalid 0-length user")
	} else {
		exists := false
		for _, u := range tc.users {
			if u.Name == msg.User {
				exists = true
			}
		}
		if !exists {
			tc.test.Errorf("Invalid user: %s", msg.User)
		}
	}
	tc.RUnlock()
	select {
	case tc.listener <- msg:
		tc.test.Logf("Message sent to robot: %v", msg)
	case <-time.After(200 * time.Millisecond):
		tc.test.Errorf("Timed out sending; user: \"%s\", channel: \"%s\", message: \"%s\"", msg.User, msg.Channel, msg.Message)
	}
}

// GetBotMessage, for tests to get replies
func (tc *TestConnector) GetBotMessage() *TestMessage {
	select {
	case incoming := <-tc.speaking:
		tc.test.Logf("Reply received from robot: %v", incoming)
		return incoming
	case <-time.After(2 * time.Second):
		tc.test.Errorf("Timed out waiting for reply in GetBotMessage")
		return nil
	}
}
