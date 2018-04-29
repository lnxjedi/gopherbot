package test

import (
	"github.com/lnxjedi/gopherbot/bot"
)

// BotMessage is for receiving messages from the robot
type BotMessage struct {
	User, Channel, Message string
	Format                 bot.MessageFormat
}

func (tc *terminalConnector) MessageHeard(u, c string) {
	return
}

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
	msg := &BotMessage{
		User:    "",
		Channel: ch,
		Message: mesg,
		Format:  f,
	}
	return tc.sendMessage(msg)
}

// SendProtocolChannelMessage sends a message to a channel
func (tc *TestConnector) SendProtocolUserChannelMessage(u, ch, mesg string, f bot.MessageFormat) (ret bot.RetVal) {
	msg := &BotMessage{
		User:    u,
		Channel: ch,
		Message: mesg,
		Format:  f,
	}
	return tc.sendMessage(msg)
}

// SendProtocolUserMessage sends a direct message to a user
func (tc *TestConnector) SendProtocolUserMessage(u string, mesg string, f bot.MessageFormat) (ret bot.RetVal) {
	msg := &BotMessage{
		User:    u,
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
