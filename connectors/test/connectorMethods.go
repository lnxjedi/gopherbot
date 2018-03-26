package test

import (
	"github.com/lnxjedi/gopherbot/bot"
)

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (tc *TestConnector) GetProtocolUserAttribute(u, attr string) (value string, ret bot.RetVal) {
	i, exists := userMap[u]
	if !exists {
		return "", bot.UserNotFound
	}
	user := tc.users[i]
	switch attr {
	case "email":
		return user.Email, bot.Ok
	case "internalID":
		return user.InternalID, bot.Ok
	case "realName", "fullName":
		return user.FullName, bot.Ok
	case "firstName":
		return user.FirstName, bot.Ok
	case "lastName":
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
	msg := &TestMessage{
		User:    "",
		Channel: ch,
		Message: mesg,
	}
	return tc.sendMessage(msg)
}

// SendProtocolChannelMessage sends a message to a channel
func (tc *TestConnector) SendProtocolUserChannelMessage(u, ch, mesg string, f bot.MessageFormat) (ret bot.RetVal) {
	msg := &TestMessage{
		User:    u,
		Channel: ch,
		Message: mesg,
	}
	return tc.sendMessage(msg)
}

// SendProtocolUserMessage sends a direct message to a user
func (tc *TestConnector) SendProtocolUserMessage(u string, mesg string, f bot.MessageFormat) (ret bot.RetVal) {
	msg := &TestMessage{
		User:    u,
		Channel: "",
		Message: mesg,
	}
	return tc.sendMessage(msg)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (tc *TestConnector) JoinChannel(c string) (ret bot.RetVal) {
	if c == "" {
		return bot.Ok
	}
	found := false
	tc.Lock()
	for _, channel := range tc.channels {
		if channel == c {
			found = true
			break
		}
	}
	if !found {
		tc.channels = append(tc.channels, c)
	}
	tc.Unlock()
	return bot.Ok
}
