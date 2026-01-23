package testutil

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// NOTE: Keep in sync with common_test.go:
// Cast of Users
const aliceID = "u0001"
const bobID = "u0002"
const carolID = "u0003"
const davidID = "u0004"
const erinID = "u0005"

func FormatIncoming(msg *robot.ConnectorMessage) string {
	uid := msg.UserID
	switch msg.UserID {
	case aliceID:
		uid = "aliceID"
	case bobID:
		uid = "bobID"
	case carolID:
		uid = "carolID"
	case davidID:
		uid = "davidID"
	case erinID:
		uid = "erinID"
	}
	logMsg := msg.MessageText
	if msg.HiddenMessage {
		logMsg = "/" + msg.MessageText
	}
	return fmt.Sprintf("%s, %s, \"%s\", %t", uid, msg.ChannelName, logMsg, msg.ThreadedMessage)
}

func FormatOutgoing(user, channel, message, thread string) string {
	printUser := user
	if user == "" {
		printUser = "null"
	} else {
		switch user {
		case aliceID:
			printUser = "alice"
		case bobID:
			printUser = "bob"
		case carolID:
			printUser = "carol"
		case davidID:
			printUser = "david"
		case erinID:
			printUser = "erin"
		}
		// Adjust for the difference in the terminal and test connectors - the
		// terminal connector applies "@user ..." when a message is sent to a user, 
		// the test connector indicates who the message was sent to in a separate field.
		msgHidden := strings.HasPrefix(message, "(") && strings.HasSuffix(message, ")")
		if msgHidden {
			message = message[1 : len(message)-1]
		}
		message = strings.TrimPrefix(message, "@"+printUser+" ")
		if msgHidden {
			message = "(" + message + ")"
		}
	}
	if channel == "" {
		channel = "null"
	}
	if strings.HasPrefix(channel, "(dm:") {
		channel = "null"
	}
	threaded := false
	if thread != "" {
		threaded = true
	}
	return fmt.Sprintf("{%s, %s, \"%s\", %t}", printUser, channel, message, threaded)
}
