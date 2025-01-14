package tbot

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
)

// NOTE: Keep in sync with common_test.go:
// Cast of Users
const alice = "alice"
const bob = "bob"
const carol = "carol"
const david = "david"
const erin = "erin"
const aliceID = "u0001"
const bobID = "u0002"
const carolID = "u0003"
const davidID = "u0004"
const erinID = "u0005"

// When the robot doesn't address the user specifically, or sends a DM
const null = ""

// ... and the Channels they play in
const general = "general"
const random = "random"
const bottest = "bottest"
const deadzone = "deadzone"

func FormatIncoming(msg *robot.ConnectorMessage) string {
	uid := msg.UserID
	switch msg.UserID {
	case aliceID:
		uid = "aliceID"
	case bobID:
		uid = "bobID"
	}
	return fmt.Sprintf("%s, %s, \"%s\", %t", uid, msg.ChannelName, msg.MessageText, msg.ThreadedMessage)
}

func FormatOutgoing(user, channel, message, thread string) string {
	if user == "" {
		user = "null"
	}
	if channel == "" {
		channel = "null"
	}
	threaded := false
	if thread != "" {
		threaded = true
	}
	return fmt.Sprintf("{%s, %s, \"%s\", %t}", user, channel, message, threaded)
}
