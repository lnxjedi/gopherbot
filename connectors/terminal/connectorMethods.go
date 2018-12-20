package terminal

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
)

func (tc *termConnector) MessageHeard(u, c string) {
	return
}

func (tc *termConnector) getUserInfo(u string) (*termUser, bool) {
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

// SetUserMap lets Gopherbot provide a mapping of usernames to user IDs
func (tc *termConnector) SetUserMap(map[string]string) {
	return
}

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (tc *termConnector) GetProtocolUserAttribute(u, attr string) (value string, ret bot.RetVal) {
	var user *termUser
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
func (tc *termConnector) SendProtocolChannelMessage(ch string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	channel := getChannel(ch)
	return tc.sendMessage(channel, msg, f)
}

// SendProtocolChannelMessage sends a message to a channel
func (tc *termConnector) SendProtocolUserChannelMessage(uid, uname, ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	channel := getChannel(ch)
	msg = "@" + uname + " " + msg
	return tc.sendMessage(channel, msg, f)
}

// SendProtocolUserMessage sends a direct message to a user
func (tc *termConnector) SendProtocolUserMessage(u string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	var user *termUser
	var exists bool
	if user, exists = tc.getUserInfo(u); !exists {
		return bot.UserNotFound
	}
	return tc.sendMessage(fmt.Sprintf("(dm:%s)", user.Name), msg, f)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
// Only useful for connectors that require it, a noop otherwise
func (tc *termConnector) JoinChannel(c string) (ret bot.RetVal) {
	return bot.Ok
}
