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

// see robot/robot.go
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
	return interfaces.SendProtocolChannelThreadMessage(channel, "", msg, r.Format, r.Incoming)
}

// see robot/robot.go
func (r Robot) SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) robot.RetVal {
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
	return interfaces.SendProtocolChannelThreadMessage(channel, thr, msg, r.Format, r.Incoming)
}

func (w *worker) SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) robot.RetVal {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	var channel string
	if ci, ok := w.maps.channel[ch]; ok {
		channel = bracket(ci.ChannelID)
	} else {
		channel = ch
	}
	return interfaces.SendProtocolChannelThreadMessage(channel, thr, msg, w.Format, w.Incoming)
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
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, "", msg, r.Format, r.Incoming)
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
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, "", msg, w.Format, w.Incoming)
}

func (r Robot) SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) robot.RetVal {
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
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, thr, msg, r.Format, r.Incoming)
}

func (w *worker) SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) robot.RetVal {
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
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, thr, msg, w.Format, w.Incoming)
}

// see robot/robot.go
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
	return interfaces.SendProtocolUserMessage(user, msg, r.Format, r.Incoming)
}

// see robot/robot.go
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
		return interfaces.SendProtocolUserMessage(user, msg, r.Format, r.Incoming)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	var thread string
	if r.Incoming.ThreadedMessage {
		thread = r.Incoming.ThreadID
	}
	w := getLockedWorker(r.tid)
	w.Unlock()
	if w.BotUser {
		return interfaces.SendProtocolChannelThreadMessage(r.Channel, thread, r.User+": "+msg, r.Format, r.Incoming)
	}
	return interfaces.SendProtocolUserChannelThreadMessage(user, r.User, r.Channel, thread, msg, r.Format, r.Incoming)
}

func (w *worker) reply(msg string, v ...interface{}) robot.RetVal {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	user := w.ProtocolUser
	if len(user) == 0 {
		user = w.User
	}
	// Support for Direct()
	if w.Channel == "" {
		return interfaces.SendProtocolUserMessage(user, msg, w.Format, w.Incoming)
	}
	var thread string
	if w.Incoming.ThreadedMessage {
		thread = w.Incoming.ThreadID
	}
	if w.BotUser {
		return interfaces.SendProtocolChannelThreadMessage(w.Channel, thread, w.User+": "+msg, w.Format, w.Incoming)
	}
	return interfaces.SendProtocolUserChannelThreadMessage(user, w.User, w.Channel, thread, msg, w.Format, w.Incoming)
}

// see robot/robot.go
func (r Robot) ReplyThread(msg string, v ...interface{}) robot.RetVal {
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
		return interfaces.SendProtocolUserMessage(user, msg, r.Format, r.Incoming)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	w := getLockedWorker(r.tid)
	w.Unlock()
	if w.BotUser {
		return interfaces.SendProtocolChannelThreadMessage(r.Channel, r.Incoming.ThreadID, r.User+": "+msg, r.Format, r.Incoming)
	}
	return interfaces.SendProtocolUserChannelThreadMessage(user, r.User, r.Channel, r.Incoming.ThreadID, msg, r.Format, r.Incoming)
}

// see robot/robot.go
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
		return interfaces.SendProtocolUserMessage(user, msg, r.Format, r.Incoming)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	var thread string
	if r.Incoming.ThreadedMessage {
		thread = r.Incoming.ThreadID
	}
	return interfaces.SendProtocolChannelThreadMessage(channel, thread, msg, r.Format, r.Incoming)
}

func (w *worker) say(msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		Log(robot.Warn, "Ignoring zero-length message in Say")
		return robot.Ok
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	// Support for Direct()
	if w.Channel == "" {
		user := w.ProtocolUser
		if len(user) == 0 {
			user = w.User
		}
		return interfaces.SendProtocolUserMessage(user, msg, w.Format, w.Incoming)
	}
	channel := w.ProtocolChannel
	if len(channel) == 0 {
		channel = w.Channel
	}
	var thread string
	if w.Incoming.ThreadedMessage {
		thread = w.Incoming.ThreadID
	}
	return interfaces.SendProtocolChannelThreadMessage(channel, thread, msg, w.Format, w.Incoming)
}

// see robot/robot.go
func (r Robot) SayThread(msg string, v ...interface{}) robot.RetVal {
	if len(msg) == 0 {
		r.Log(robot.Warn, "Ignoring zero-length message in SayThread")
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
		return interfaces.SendProtocolUserMessage(user, msg, r.Format, r.Incoming)
	}
	channel := r.ProtocolChannel
	if len(channel) == 0 {
		channel = r.Channel
	}
	return interfaces.SendProtocolChannelThreadMessage(channel, r.Incoming.ThreadID, msg, r.Format, r.Incoming)
}
