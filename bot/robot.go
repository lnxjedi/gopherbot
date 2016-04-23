package bot

import (
	"encoding/json"
	"fmt"
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

func (b *robot) CheckAdmin(user string) bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for _, adminUser := range b.adminUsers {
		if user == adminUser {
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

// SendXXXMessage functions exist so plugin writers don't need
// to pass a format var for every message, when a Variable font is
// wanted 99% of the time. It's easy to get Fixed, though, using
// the convenience function, or by manually setting r.Format.
func (r Robot) SendChannelMessage(ch, msg string) {
	r.SendProtocolChannelMessage(ch, msg, r.Format)
}

func (r Robot) SendUserChannelMessage(u, ch, msg string) {
	r.SendProtocolUserChannelMessage(u, ch, msg, r.Format)
}

func (r Robot) SendUserMessage(u, msg string) {
	r.SendProtocolUserMessage(u, msg, r.Format)
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
