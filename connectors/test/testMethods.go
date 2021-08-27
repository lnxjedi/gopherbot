package test

import (
	"errors"
	"time"
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
			tc.test.Errorf("Invalid channel: %s", msg.Channel)
		}
	}
	if msg.User == "" {
		tc.test.Errorf("Invalid 0-length user")
	} else {
		exists := false
		for _, u := range tc.users {
			if u.InternalID == msg.User {
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

// GetBotMessage for tests to get replies
func (tc *TestConnector) GetBotMessage() (*TestMessage, error) {
	select {
	case incoming := <-tc.speaking:
		message := incoming.Message
		if len(incoming.Message) > 16 {
			message = incoming.Message[0:16] + " ..."
		}
		tc.test.Logf("Reply received from robot: u:%s, c:%s, m:%s", incoming.User, incoming.Channel, message)
		time.Sleep(100 * time.Millisecond)
		return incoming, nil
	case <-time.After(4 * time.Second):
		return nil, errors.New("timeout waiting for reply from robot")
	}
}
