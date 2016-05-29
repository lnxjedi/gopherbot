package bot

import (
	"fmt"
	"reflect"
	"time"

	otp "github.com/dgryski/dgoogauth"
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

// CheckOTP returns true if the provided string is a valid OTP code for the user.
// See the builtInlaunchcodes.go plugin.
func (r Robot) CheckOTP(code string) bool {
	otpKey := "bot:OTP:" + r.User
	var userOTP otp.OTPConfig
	updated := false
	lock, exists, err := r.checkoutDatum(otpKey, &userOTP, true)
	if err != nil {
		return false
	}
	defer func() {
		if updated {
			err := r.updateDatum(otpKey, lock, &userOTP)
			if err != nil {
				r.Log(Error, fmt.Errorf("Saving OTP config: %v", err))
			}
		} else {
			// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
			// in case there's another thread waiting.
			r.checkin(otpKey, lock)
		}
	}()
	if !exists {
		return false
	}
	valid, err := userOTP.Authenticate(code)
	if err != nil {
		r.Log(Error, fmt.Errorf("Problem authenticating launch code: %v", err))
		return false
	}
	if valid {
		return true
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

// Pause is a convenience function to pause some fractional number of seconds.
func (r Robot) Pause(s float64) {
	ms := time.Duration(s * float64(1000))
	time.Sleep(ms * time.Millisecond)
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

// GetBotAttribute returns an attribute of the robot or "" if unknown.
// Current attributes:
// name, alias, fullName, contact
func (r Robot) GetBotAttribute(a string) string {
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
	case "email":
		return b.email
	case "contact", "admin", "adminContact":
		return b.adminContact
	}
	return ""
}

// GetUserAttribute returns an attribute of a user or "" if unknown.
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone
// TODO: supplement data with gopherbot.json user's table
func (r Robot) GetUserAttribute(u, a string) string {
	attr, _ := r.GetProtocolUserAttribute(u, a)
	return attr
}

// GetSenderAttribute returns an attribute of the sending user or "" if unknown.
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone
// TODO: supplement data with gopherbot.json user's table
func (r Robot) GetSenderAttribute(a string) string {
	attr, _ := r.GetProtocolUserAttribute(r.User, a)
	return attr
}

// GetPluginConfig will unmarshall the plugin's Config section into
// a provided struct.
func (r Robot) GetPluginConfig(dptr interface{}) bool {
	b := r.robot
	b.lock.RLock()
	defer b.lock.RUnlock()
	plugin := b.plugins[b.plugIDmap[r.pluginID]]
	tp := reflect.ValueOf(dptr)
	if tp.Kind() != reflect.Ptr {
		return false
	}
	p := reflect.Indirect(tp)
	if p.Kind() != reflect.Ptr {
		return false
	}
	if p.Type() != reflect.ValueOf(plugin.config).Type() {
		return false
	}
	p.Set(reflect.ValueOf(plugin.config))
	return true
}

// WaitForReply lets a plugin temporarily register a regex for a reply
// expected to an multi-step command, e.g. sending an email. An error
// is returned if the user already has a multi-step command in progress
// in the given channel, or if the regex id is wrong. Otherwise, any
// reply is returned with matched indicating whether the reply matched
// the regex. If the timeout is reached, timedOut is true and the reply is "".
func (r Robot) WaitForReply(regexId string, timeout int) (matched, timedOut bool, replyText string, err error) {
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
		return false, false, "", err
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
		return false, false, "", err
	}
	rep.replyChannel = make(chan reply)
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
		// matched=false, timedOut=true
		return false, true, "", err
	case replied, _ := <-rep.replyChannel:
		// Note: the replies[] entry is deleted in handleMessage
		return replied.matched, false, replied.rep, nil
	}
}

// SendChannelMessage lets a plugin easily send a message to an arbitrary
// channel. Use Robot.Fixed().SencChannelMessage(...) for fixed-width
// font.
func (r Robot) SendChannelMessage(channel, msg string) {
	r.SendProtocolChannelMessage(channel, msg, r.Format)
}

// SendUserChannelMessage lets a plugin easily send a message directed to
// a specific user in a specific channel without fiddling with the robot
// object. Use Robot.Fixed().SencChannelMessage(...) for fixed-width
// font.
func (r Robot) SendUserChannelMessage(user, channel, msg string) {
	r.SendProtocolUserChannelMessage(user, channel, msg, r.Format)
}

// SendUserMessage lets a plugin easily send a DM to a user. If a DM
// isn't possible, the connector should message the user in a channel.
func (r Robot) SendUserMessage(user, msg string) {
	r.SendProtocolUserMessage(user, msg, r.Format)
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
