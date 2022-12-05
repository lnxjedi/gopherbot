package bot

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
)

/* send_message.go - all the message sending methods for a worker or a Robot.
The worker methods are used by the worker during message processing, before
any calls to makeRobot(). The main difference between the worker and Robot
versions is that the Robot version uses it's local copy of the robot.Message,
which may have been modified by e.g. r.Direct(), r.Fixed(), etc.
*/

// messageHeard sends a typing notification
func (r Robot) messageHeard() {
	user := r.ProtocolUser
	if len(user) == 0 {
		user = r.User
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	interfaces.MessageHeard(user, channel)
}

func (w *worker) messageHeard() {
	user := w.ProtocolUser
	if len(user) == 0 {
		user = w.User
	}
	channel := w.ProtocolChannel
	if len(channel) == 0 {
		channel = w.Channel
	}
	interfaces.MessageHeard(user, channel)
}

// SendChannelMessage lets a plugin easily send a message to an arbitrary
// channel. Use Robot.Fixed().SendChannelMessage(...) for fixed-width
// font.
func (r Robot) SendChannelMessage(ch, msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in SendChannelMessage")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	var channel string
	if ci, ok := r.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return interfaces.SendProtocolChannelMessage(channel, msg, r.Format)
}

func (w *worker) SendChannelMessage(ch, msg string, v ...interface{}) robot.RetVal {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	var channel string
	if ci, ok := w.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return interfaces.SendProtocolChannelMessage(channel, msg, w.Format)
}

// SendUserChannelMessage lets a plugin easily send a message directed to
// a specific user in a specific channel without fiddling with the robot
// object. Note that this will fail with UserNotFound if the connector
// can't resolve usernames, or the username isn't mapped to a user ID in
// the UserRoster.
func (r Robot) SendUserChannelMessage(u, ch, msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in SendUserChannelMessage")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	var user string
	if ui, ok := r.maps.user[u]; ok {
		user = bracket(ui.UserID)
	} else {
		user = u
	}
	var channel string
	if ci, ok := r.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return interfaces.SendProtocolUserChannelMessage(user, u, channel, msg, r.Format)
}

func (w *worker) SendUserChannelMessage(u, ch, msg string, v ...interface{}) robot.RetVal {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	var user string
	if ui, ok := w.maps.user[u]; ok {
		user = bracket(ui.UserID)
	} else {
		user = u
	}
	var channel string
	if ci, ok := w.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return interfaces.SendProtocolUserChannelMessage(user, u, channel, msg, w.Format)
}

// SendUserMessage lets a plugin easily send a DM to a user. If a DM
// fails, an error should be returned, since DMs may be used for sending
// secret/sensitive information.
func (r Robot) SendUserMessage(u, msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in SendUserMessage")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	var user string
	if ui, ok := r.maps.user[u]; ok {
		user = bracket(ui.UserID)
	} else {
		user = u
	}
	return interfaces.SendProtocolUserMessage(user, msg, r.Format)
}

func (w *worker) SendUserMessage(u, msg string, v ...interface{}) robot.RetVal {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	var user string
	if ui, ok := w.maps.user[u]; ok {
		user = bracket(ui.UserID)
	} else {
		user = u
	}
	return interfaces.SendProtocolUserMessage(user, msg, w.Format)
}

// Reply directs a message to the user
func (r Robot) Reply(msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in Reply")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	user := r.ProtocolUser
	if len(user) == 0 {
		user = r.User
	}
	// Support for Direct()
	if r.Channel == "" {
		return interfaces.SendProtocolUserMessage(user, msg, r.Format)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	w := getLockedWorker(r.tid)
	w.Unlock()
	if w.BotUser {
		return interfaces.SendProtocolChannelMessage(r.Channel, r.User+": "+msg, r.Format)
	}
	return interfaces.SendProtocolUserChannelMessage(user, r.User, r.Channel, msg, r.Format)
}

func (w *worker) Reply(msg string, v ...interface{}) robot.RetVal {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	user := w.ProtocolUser
	if len(user) == 0 {
		user = w.User
	}
	// Support for Direct()
	if w.Channel == "" {
		return interfaces.SendProtocolUserMessage(user, msg, w.Format)
	}
	channel := w.ProtocolChannel
	if len(channel) == 0 {
		channel = w.Channel
	}
	if w.BotUser {
		return interfaces.SendProtocolChannelMessage(w.Channel, w.User+": "+msg, w.Format)
	}
	return interfaces.SendProtocolUserChannelMessage(user, w.User, w.Channel, msg, w.Format)
}

// Say just sends a message to the user or channel
func (r Robot) Say(msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in Say")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	// Support for Direct()
	if r.Channel == "" {
		user := r.ProtocolUser
		if len(user) == 0 {
			user = r.User
		}
		return interfaces.SendProtocolUserMessage(user, msg, r.Format)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	return interfaces.SendProtocolChannelMessage(channel, msg, r.Format)
}

func (w *worker) Say(msg string, v ...interface{}) robot.RetVal {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	// Support for Direct()
	if w.Channel == "" {
		user := w.ProtocolUser
		if len(user) == 0 {
			user = w.User
		}
		return interfaces.SendProtocolUserMessage(user, msg, w.Format)
	}
	channel := w.ProtocolChannel
	if len(channel) == 0 {
		channel = w.Channel
	}
	return interfaces.SendProtocolChannelMessage(channel, msg, w.Format)
}
