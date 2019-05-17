package rocket

import (
	"github.com/lnxjedi/gopherbot/bot"
)

func (rc *rocketConnector) MessageHeard(u, c string) {
	return
}

// SetUserMap lets Gopherbot provide a mapping of usernames to user IDs
func (rc *rocketConnector) SetUserMap(m map[string]string) {
	rc.Lock()
	rc.gbuserMap = m
	rc.Unlock()
	return
}

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (rc *rocketConnector) GetProtocolUserAttribute(u, attr string) (value string, ret bot.RetVal) {
	return "", bot.Ok
}

// SendProtocolChannelMessage sends a message to a channel
func (rc *rocketConnector) SendProtocolChannelMessage(ch string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	return rc.sendMessage(ch, msg)
}

// SendProtocolChannelMessage sends a message to a channel
func (rc *rocketConnector) SendProtocolUserChannelMessage(uid, uname, ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
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
			rc.Log(bot.Warn, "Unable to resolve a rocket chat username for %s", uid)
		}
	}
	if len(user) > 0 {
		msg = "@" + uname + " " + msg
	}
	return rc.sendMessage(ch, msg)
}

// SendProtocolUserMessage sends a direct message to a user
func (rc *rocketConnector) SendProtocolUserMessage(u string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	var dchan, uid string
	found := false
	uid, found = bot.ExtractID(u)
	rc.RLock()
	if !found {
		uid, found = rc.userNameIDMap[u]
	}
	if !found {
		rc.RUnlock()
		return bot.UserNotFound
	}
	dchan, found = rc.userDM[uid]
	if !found {
		rc.Log(bot.Error, "unable to locate DM channel for %s", u)
		rc.RUnlock()
		return bot.FailedMessageSend
	}
	rc.RUnlock()
	// sendMessage expects internal channels IDs to be bracketed
	return rc.sendMessage("<"+dchan+">", msg)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
// Only useful for connectors that require it, a noop otherwise
func (rc *rocketConnector) JoinChannel(c string) (ret bot.RetVal) {
	rc.Lock()
	rid, ok := rc.channelIDs[c]
	if !ok {
		var err error
		rid, err = rc.rt.GetChannelId(c)
		if err != nil {
			rc.Log(bot.Error, "getting channel ID joining channel %s: %v", c, err)
			rc.Unlock()
			return bot.ChannelNotFound
		}
		rc.channelIDs[c] = rid
		rc.channelNames[rid] = c
	}
	if _, ok := rc.joinedChannels[rid]; !ok {
		rc.joinedChannels[rid] = struct{}{}
		if err := rc.rt.JoinChannel(rid); err != nil {
			rc.Log(bot.Error, "joining channel %s/%s: %v", c, rid, err)
		}
	}
	rc.Unlock()
	return bot.Ok
}
