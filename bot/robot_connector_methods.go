package bot

import "strings"

// GetUserAttribute returns a AttrRet with
// - The string Attribute of a user, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: supplement data with gopherbot.yaml user's table, if an
// admin wants to supplment whats available from the protocol.
func (r *Robot) GetUserAttribute(u, a string) *AttrRet {
	a = strings.ToLower(a)
	c := r.getContext()
	var user string
	if ui, ok := c.maps.user[u]; ok {
		user = "<" + ui.UserID + ">"
	} else {
		user = u
	}
	attr, ret := botCfg.GetProtocolUserAttribute(user, a)
	return &AttrRet{attr, ret}
}

// messageHeard sends a typing notification
func (c *botContext) messageHeard() {
	user := c.ProtocolUser
	if len(user) == 0 {
		user = c.User
	}
	channel := c.ProtocolChannel
	if len(channel) == 0 {
		channel = c.Channel
	}
	botCfg.MessageHeard(user, channel)
}

// GetSenderAttribute returns a AttrRet with
// - The string Attribute of the sender, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: (see above)
func (r *Robot) GetSenderAttribute(a string) *AttrRet {
	a = strings.ToLower(a)
	switch a {
	case "name", "username", "handle", "user", "user name":
		return &AttrRet{r.User, Ok}
	default:
		user := r.ProtocolUser
		if len(user) == 0 {
			user = r.User
		}
		attr, ret := botCfg.GetProtocolUserAttribute(user, a)
		return &AttrRet{attr, ret}
	}
}

// SendChannelMessage lets a plugin easily send a message to an arbitrary
// channel. Use Robot.Fixed().SendChannelMessage(...) for fixed-width
// font.
func (r *Robot) SendChannelMessage(ch, msg string) RetVal {
	c := r.getContext()
	var channel string
	if ci, ok := c.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return botCfg.SendProtocolChannelMessage(channel, msg, r.Format)
}

// SendUserChannelMessage lets a plugin easily send a message directed to
// a specific user in a specific channel without fiddling with the robot
// object. Note that this will fail with UserNotFound if the connector
// can't resolve usernames, or the username isn't mapped to a user ID in
// the UserRoster.
func (r *Robot) SendUserChannelMessage(u, ch, msg string) RetVal {
	c := r.getContext()
	var user string
	if ui, ok := c.maps.user[u]; ok {
		user = bracket(ui.UserID)
	} else {
		user = u
	}
	var channel string
	if ci, ok := c.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return botCfg.SendProtocolUserChannelMessage(user, u, channel, msg, r.Format)
}

// SendUserMessage lets a plugin easily send a DM to a user. If a DM
// isn't possible, the connector should message the user in a channel.
func (r *Robot) SendUserMessage(u, msg string) RetVal {
	c := r.getContext()
	var user string
	if ui, ok := c.maps.user[u]; ok {
		user = bracket(ui.UserID)
	} else {
		user = u
	}
	return botCfg.SendProtocolUserMessage(user, msg, r.Format)
}

// Reply directs a message to the user
func (r *Robot) Reply(msg string) RetVal {
	user := r.ProtocolUser
	if len(user) == 0 {
		user = r.User
	}
	// Support for Direct()
	if r.Channel == "" {
		return botCfg.SendProtocolUserMessage(user, msg, r.Format)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	return botCfg.SendProtocolUserChannelMessage(user, r.User, r.Channel, msg, r.Format)
}

// Say just sends a message to the user or channel
func (r *Robot) Say(msg string) RetVal {
	// Support for Direct()
	if r.Channel == "" {
		user := r.ProtocolUser
		if len(user) == 0 {
			user = r.User
		}
		return botCfg.SendProtocolUserMessage(user, msg, r.Format)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	return botCfg.SendProtocolChannelMessage(channel, msg, r.Format)
}
