package bot

// Robot is passed to the plugin to enable convenience functions Say and Reply
type Robot struct {
	User     string        // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel  string        // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Format   MessageFormat // The outgoing message format, one of Fixed or Variable
	pluginID string        // Pass the ID in for later identificaton of the plugin
	Gobot
}

/* robot.go defines some convenience functions on struct Robot to
   simplify use by plugins. */

// Fixed is a convenience function for sending a message with fixed width
// font. e.g. r.Reply(xxx) replies in variable width font, but
// r.Fixed().Reply(xxx) replies in a fixed-width font.
func (r Robot) Fixed() Robot {
	r.Format = Fixed
	return r
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
