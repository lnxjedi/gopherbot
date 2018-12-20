package test

import (
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
)

// BotMessage is for receiving messages from the robot
type BotMessage struct {
	User, Channel, Message string
	Format                 bot.MessageFormat
}

func (tc *TestConnector) getUserInfo(u string) (*testUser, bool) {
	var i int
	var exists bool
	if id, ok := bot.ExtractID(u); ok {
		i, exists = userIDMap[id]
	} else {
		i, exists = userMap[u]
	}
	if exists {
		return &tc.users[i], true
	}
	return nil, false
}

func getChannel(c string) string {
	if ch, ok := bot.ExtractID(c); ok {
		return strings.TrimPrefix(ch, "#")
	}
	return c
}

// MessageHeard indicates to the user a message was heard;
// for test/terminal it's a noop.
func (tc *TestConnector) MessageHeard(u, c string) {
	return
}

// GetProtocolUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (tc *TestConnector) GetProtocolUserAttribute(u, attr string) (value string, ret bot.RetVal) {
	var user *testUser
	var exists bool
	if user, exists = tc.getUserInfo(u); !exists {
		return "", bot.UserNotFound
	}
	switch attr {
	case "email":
		return user.Email, bot.Ok
	case "internalid":
		return user.InternalID, bot.Ok
	case "realname", "fullname", "real name", "full name":
		return user.FullName, bot.Ok
	case "firstname", "first name":
		return user.FirstName, bot.Ok
	case "lastname", "last name":
		return user.LastName, bot.Ok
	case "phone":
		return user.Phone, bot.Ok
	// that's all the attributes we can currently get from slack
	default:
		return "", bot.AttributeNotFound
	}
}

// SendProtocolChannelMessage sends a message to a channel
func (tc *TestConnector) SendProtocolChannelMessage(ch string, mesg string, f bot.MessageFormat) (ret bot.RetVal) {
	channel := getChannel(ch)
	msg := &BotMessage{
		User:    "",
		Channel: channel,
		Message: mesg,
		Format:  f,
	}
	return tc.sendMessage(msg)
}

// SendProtocolUserChannelMessage sends a message to a user in a channel
func (tc *TestConnector) SendProtocolUserChannelMessage(uid, uname, ch, mesg string, f bot.MessageFormat) (ret bot.RetVal) {
	channel := getChannel(ch)
	msg := &BotMessage{
		User:    uname,
		Channel: channel,
		Message: mesg,
		Format:  f,
	}
	return tc.sendMessage(msg)
}

// SendProtocolUserMessage sends a direct message to a user
func (tc *TestConnector) SendProtocolUserMessage(u string, mesg string, f bot.MessageFormat) (ret bot.RetVal) {
	var user *testUser
	var exists bool
	if user, exists = tc.getUserInfo(u); !exists {
		return bot.UserNotFound
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
func (tc *TestConnector) JoinChannel(c string) (ret bot.RetVal) {
	return bot.Ok
}
