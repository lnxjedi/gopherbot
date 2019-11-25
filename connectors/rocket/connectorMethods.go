package rocket

import "github.com/lnxjedi/gopherbot/robot"

func (rc *rocketConnector) MessageHeard(u, c string) {
	return
}

// SetUserMap lets Gopherbot provide a mapping of usernames to user IDs
func (rc *rocketConnector) SetUserMap(n map[string]string) {
	m := make(map[string]string)
	for name, id := range n {
		m[id] = name
	}
	rc.Lock()
	rc.gbuserNameIDMap = n
	rc.gbuserIDNameMap = m
	rc.Unlock()
	return
}

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (rc *rocketConnector) GetProtocolUserAttribute(u, attr string) (value string, ret robot.RetVal) {
	return "", robot.Ok
}

// SendProtocolChannelMessage sends a message to a channel
func (rc *rocketConnector) SendProtocolChannelMessage(ch string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	return rc.sendMessage(ch, formatMessage(msg, f))
}

// SendProtocolChannelMessage sends a message to a channel
func (rc *rocketConnector) SendProtocolUserChannelMessage(uid, uname, ch, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	var user string
	// We prefer to use @(rocketchat username), looked up from
	// the user ID.
	if len(uid) > 0 {
		rc.RLock()
		if u, ok := rc.userIDNameMap[uid]; ok {
			user = u
		}
		rc.RUnlock()
	}
	if len(user) == 0 {
		if len(uname) > 0 {
			user = uname
		} else {
			rc.Log(robot.Warn, "Unable to resolve a rocket chat username for %s", uid)
		}
	}
	msg = formatMessage(msg, f)
	if len(user) > 0 {
		msg = "@" + uname + " " + msg
	}
	return rc.sendMessage(ch, msg)
}

// SendProtocolUserMessage sends a direct message to a user
func (rc *rocketConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	var uid, dchan, user string
	var ok bool
	var err error
	if uid, ok = rc.ExtractID(u); ok {
		rc.RLock()
		user, ok = rc.userIDNameMap[uid]
		rc.RUnlock()
		if !ok {
			return robot.UserNotFound
		}
	} else {
		user = u
	}
	rc.RLock()
	dchan, ok = rc.userDM[user]
	rc.RUnlock()
	if !ok {
		if dchan, err = rc.rt.CreateDirectMessage(user); err != nil {
			rc.Log(robot.Error, "creating direct message for %s: %v", user, err)
			return robot.FailedMessageSend
		}
		rc.Lock()
		rc.userDM[user] = dchan
		rc.Unlock()
	}
	// sendMessage expects internal channels IDs to be bracketed
	return rc.sendMessage("<"+dchan+">", formatMessage(msg, f))
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
// Only useful for connectors that require it, a noop otherwise
func (rc *rocketConnector) JoinChannel(c string) (ret robot.RetVal) {
	rc.Lock()
	rid, ok := rc.channelIDs[c]
	if !ok {
		var err error
		rid, err = rc.rt.GetChannelId(c)
		if err != nil {
			rc.Log(robot.Error, "getting channel ID joining channel %s: %v", c, err)
			rc.Unlock()
			return robot.ChannelNotFound
		}
		rc.channelIDs[c] = rid
		rc.channelNames[rid] = c
	}
	if _, ok := rc.joinedChannels[rid]; !ok {
		rc.joinedChannels[rid] = struct{}{}
		if err := rc.rt.JoinChannel(rid); err != nil {
			rc.Log(robot.Error, "joining channel %s/%s: %v", c, rid, err)
		}
	}
	rc.Unlock()
	return robot.Ok
}
