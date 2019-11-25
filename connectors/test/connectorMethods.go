package test

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// BotMessage is for receiving messages from the robot
type BotMessage struct {
	User, Channel, Message string
	Format                 robot.MessageFormat
}

func (tc *TestConnector) getUserInfo(u string) (*testUser, bool) {
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

func (tc *TestConnector) getChannel(c string) string {
	if ch, ok := tc.ExtractID(c); ok {
		return strings.TrimPrefix(ch, "#")
	}
	return c
}

// MessageHeard indicates to the user a message was heard;
// for test/terminal it's a noop.
func (tc *TestConnector) MessageHeard(u, c string) {
	return
}

// SetUserMap lets Gopherbot provide a mapping of usernames to user IDs
func (tc *TestConnector) SetUserMap(map[string]string) {
	return
}

// GetProtocolUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (tc *TestConnector) GetProtocolUserAttribute(u, attr string) (value string, ret robot.RetVal) {
	var user *testUser
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
func (tc *TestConnector) SendProtocolChannelMessage(ch string, mesg string, f robot.MessageFormat) (ret robot.RetVal) {
	channel := tc.getChannel(ch)
	msg := &BotMessage{
		User:    "",
		Channel: channel,
		Message: mesg,
		Format:  f,
	}
	return tc.sendMessage(msg)
}

// SendProtocolUserChannelMessage sends a message to a user in a channel
func (tc *TestConnector) SendProtocolUserChannelMessage(uid, uname, ch, mesg string, f robot.MessageFormat) (ret robot.RetVal) {
	channel := tc.getChannel(ch)
	msg := &BotMessage{
		User:    uname,
		Channel: channel,
		Message: mesg,
		Format:  f,
	}
	return tc.sendMessage(msg)
}

// SendProtocolUserMessage sends a direct message to a user
func (tc *TestConnector) SendProtocolUserMessage(u string, mesg string, f robot.MessageFormat) (ret robot.RetVal) {
	var user *testUser
	var exists bool
	if user, exists = tc.getUserInfo(u); !exists {
		return robot.UserNotFound
	}
	msg := &BotMessage{
		User:    user.Name,
		Channel: "",
		Message: mesg,
		Format:  f,
	}
	return tc.sendMessage(msg)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
// Only useful for connectors that require it, a noop otherwise
func (tc *TestConnector) JoinChannel(c string) (ret robot.RetVal) {
	return robot.Ok
}
