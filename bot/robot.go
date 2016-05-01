package bot

import (
	"encoding/json"
	"fmt"
	"time"
)

type MessageFormat int

// Outgoing message format, Variable or Fixed
const (
	Variable MessageFormat = iota // variable font width
	Fixed
)

// Robot is passed to the plugin to enable convenience functions Say and Reply
type Robot struct {
	User     string        // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel  string        // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Format   MessageFormat // The outgoing message format, one of Fixed or Variable
	pluginID string        // Pass the ID in for later identificaton of the plugin
	*robot                 // Add a pointer to the robot for it's public methods, which includes the Connector provided by e.g. slack
}

/* robot.go defines some convenience functions on struct Robot to
   simplify use by plugins. */

// CheckAdmin returns true if the user is a configured administrator of the robot.
func (r Robot) CheckAdmin() bool {
	b := r.robot
	b.lock.RLock()
	defer b.lock.RUnlock()
	for _, adminUser := range b.adminUsers {
		if r.User == adminUser {
			return true
		}
	}
	return false
}

// Fixed is a convenience function for sending a message with fixed width
// font. e.g. r.Reply(xxx) replies in variable width font, but
// r.Fixed().Reply(xxx) replies in a fixed-width font.
func (r Robot) Fixed() Robot {
	r.Format = Fixed
	return r
}

// RandomString is a convenience function for returning a random string
// from a slice of strings, so that replies can vary.
func (r Robot) RandomString(s []string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	return s[random.Intn(l)]
}

// RandomInt uses the robot's seeded random to return a random int 0 <= retval < n
func (r Robot) RandomInt(n int) int {
	return random.Intn(n)
}

// GetAttribute returns an attribute of the robot or "" if unknown.
// Current attributes:
// name, alias, fullName, contact
func (r Robot) GetAttribute(a string) string {
	b := r.robot
	b.lock.RLock()
	defer b.lock.RUnlock()
	switch a {
	case "name":
		return b.name
	case "fullName", "realName":
		return b.fullName
	case "alias":
		return string(b.alias)
	case "contact", "admin", "adminContact":
		return b.adminContact
	}
	return ""
}

// GetUserAttribute returns an attribute of a user or "" if unknown.
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone
// TODO: supplement data with gopherbot.json user's table
func (r Robot) GetUserAttribute(a string) string {
	attr, _ := r.GetProtocolUserAttribute(r.User, a)
	return attr
}

// GetPluginConfig will unmarshall the plugin's Config section into
// a provided struct.
func (r Robot) GetPluginConfig(v interface{}) error {
	b := r.robot
	b.lock.RLock()
	defer b.lock.RUnlock()
	plugin := b.plugins[b.plugIDmap[r.pluginID]]
	err := json.Unmarshal(plugin.Config, v)
	if err != nil {
		b.Log(Error, fmt.Errorf("Unmarshaling plugin config for %s: %v", plugin.Name, err))
	}
	return err
}

// WaitForReply lets a plugin temporarily register a regex for a reply
// expected to an multi-step command, e.g. sending an email. An error
// is returned if the user already has a multi-step command in progress
// in the given channel, or if the timeout expires. If needCommand is true,
// the reply must be directed at the robot.
func (r Robot) WaitForReply(regexId string, timeout int, needCommand bool) (string, error) {
	matcher := replyMatcher{
		user:    r.User,
		channel: r.Channel,
	}
	// We don't immediately defer an unlock because this function blocks on the
	// reply channel - so we need to Unlock() at every error return point.
	botLock.Lock()
	// See if there's already a continuation in progress for this Robot:user,channel,
	rep, exists := replies[matcher]
	if exists {
		err := fmt.Errorf("A reply is already being waited on for user %s in channel %s", r.User, r.Channel)
		r.Log(Warn, err)
		botLock.Unlock()
		return "", err
	}
	b := r.robot
	b.lock.RLock()
	plugin := b.plugins[b.plugIDmap[r.pluginID]]
	plugName := plugin.Name
	for _, matcher := range plugin.ReplyMatchers {
		if matcher.Command == regexId {
			rep.regex = matcher.Regex
			// Copy the regex - if a reload happens while waiting for a reply, a pointer could invalidate
			rep.re = matcher.re.Copy()
			break
		}
	}
	b.lock.RUnlock()
	if rep.re == nil {
		err := fmt.Errorf("Unable to resolve a reply matcher for plugin %s, regexID %s", plugin.Name, regexId)
		r.Log(Error, err)
		botLock.Unlock()
		return "", err
	}
	rep.reply = make(chan string)
	rep.needCommand = needCommand
	r.Log(Trace, fmt.Sprintf("Adding matcher to replies: %q", matcher))
	replies[matcher] = rep
	// Now that we've added the reply to the map, unlock the bot so we can block
	// on the channel for a reply.
	botLock.Unlock()
	// Start a goroutine to delete the reply request if it still exists after a minute.
	// If it's matched in the meantime, it should get deleted at that point.
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		err := fmt.Errorf("Plugin \"%s\" timed out waiting for a reply to regex \"%s\"", plugName, regexId)
		b.Log(Warn, err)
		botLock.Lock()
		// reply timed out, free up this matcher for later reply requests
		delete(replies, matcher)
		botLock.Unlock()
		return "", err
	case reply := <-rep.reply:
		// Note: the replies[] entry is deleted in handleMessage
		return reply, nil
	}
}

// SendXXXMessage functions exist so plugin writers don't need
// to pass a format var for every message, when a Variable font is
// wanted 99% of the time. It's easy to get Fixed, though, using
// the convenience function, or by manually setting r.Format.
func (r Robot) SendChannelMessage(msg string) {
	r.SendProtocolChannelMessage(r.Channel, msg, r.Format)
}

func (r Robot) SendUserChannelMessage(msg string) {
	r.SendProtocolUserChannelMessage(r.User, r.Channel, msg, r.Format)
}

func (r Robot) SendUserMessage(msg string) {
	r.SendProtocolUserMessage(r.User, msg, r.Format)
}

// Reply directs a message to the user
func (r Robot) Reply(msg string) {
	if r.Channel == "" {
		r.SendProtocolUserMessage(r.User, msg, r.Format)
	} else {
		r.SendProtocolUserChannelMessage(r.User, r.Channel, msg, r.Format)
	}
}

// Say just sends a message to the user or channel
func (r Robot) Say(msg string) {
	if r.Channel == "" {
		r.SendProtocolUserMessage(r.User, msg, r.Format)
	} else {
		r.SendProtocolChannelMessage(r.Channel, msg, r.Format)
	}
}
