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

// tryResolveUser resolves the user to its internal ID wrapped in brackets if available.
// If the user is not found in the internal map, it returns the original username.
// This allows the chat connector to handle unresolved usernames appropriately.
func (r *Robot) tryResolveUser(u string) string {
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	if pm, ok := r.maps.userProto[protocol]; ok {
		if ui, ok := pm[u]; ok {
			return bracket(ui.UserID)
		}
	}
	return u
}

func (r *Robot) tryResolveUserForProtocol(protocol, u string) string {
	p := normalizeProtocolName(protocol)
	if p != "" {
		if pm, ok := r.maps.userProto[p]; ok {
			if ui, ok := pm[u]; ok {
				return bracket(ui.UserID)
			}
		}
	}
	return u
}

func (w *worker) tryResolveUser(u string) string {
	protocol := protocolFromIncoming(w.Incoming, w.Protocol)
	if pm, ok := w.maps.userProto[protocol]; ok {
		if ui, ok := pm[u]; ok {
			return bracket(ui.UserID)
		}
	}
	return u
}

// tryResolveChannel resolves the channel to its internal ID wrapped in brackets if available.
// If the channel is not found in the internal map, it returns the original channel name.
// This allows the chat connector to handle unresolved channels appropriately.
func (r *Robot) tryResolveChannel(ch string) string {
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	if ci, ok := getProtocolChannelByName(r.maps, protocol, ch); ok {
		return bracket(ci.ChannelID)
	}
	return ch
}

func (r *Robot) tryResolveChannelForProtocol(protocol, ch string) string {
	if ci, ok := getProtocolChannelByName(r.maps, protocol, ch); ok {
		return bracket(ci.ChannelID)
	}
	return ch
}

func (w *worker) tryResolveChannel(ch string) string {
	protocol := protocolFromIncoming(w.Incoming, w.Protocol)
	if ci, ok := getProtocolChannelByName(w.maps, protocol, ch); ok {
		return bracket(ci.ChannelID)
	}
	return ch
}

// prepareMessage validates and formats the message.
// It returns the formatted message and a boolean indicating whether the message was empty.
// If the message is empty, it logs a warning and returns true.
func (r *Robot) prepareMessage(fn, msg string, v ...interface{}) (string, bool) {
	if len(msg) == 0 {
		w := getLockedWorker(r.tid)
		w.Unlock()

		w.Log(robot.Warn, "%s: Ignoring zero-length message", fn)
		return "", true
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	return msg, false
}

func (w *worker) prepareMessage(fn, msg string, v ...interface{}) (string, bool) {
	if len(msg) == 0 {
		w.Log(robot.Warn, "%s: Ignoring zero-length message", fn)
		return "", true
	}
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	return msg, false
}

func protocolIncoming(in *robot.ConnectorMessage, protocol string) *robot.ConnectorMessage {
	p := normalizeProtocolName(protocol)
	if in == nil {
		return &robot.ConnectorMessage{Protocol: p}
	}
	msg := *in
	msg.Protocol = p
	return &msg
}

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
	conn := getConnectorForProtocol(protocolFromIncoming(r.Incoming, r.Protocol))
	if conn != nil {
		conn.MessageHeard(user, channel)
	}
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
	conn := getConnectorForProtocol(protocolFromIncoming(w.Incoming, w.Protocol))
	if conn != nil {
		conn.MessageHeard(user, channel)
	}
}

// see robot/robot.go
func (r Robot) SendChannelMessage(ch, msg string, v ...interface{}) robot.RetVal {
	msg, empty := r.prepareMessage("SendChannelMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	channel := r.tryResolveChannel(ch)
	return interfaces.SendProtocolChannelThreadMessage(channel, "", msg, r.Format, r.Incoming)
}

// see robot/robot.go
func (r Robot) SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) robot.RetVal {
	msg, empty := r.prepareMessage("SendChannelThreadMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	channel := r.tryResolveChannel(ch)
	return interfaces.SendProtocolChannelThreadMessage(channel, thr, msg, r.Format, r.Incoming)
}

func (w *worker) SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) robot.RetVal {
	msg, empty := w.prepareMessage("SendChannelThreadMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	channel := w.tryResolveChannel(ch)
	return interfaces.SendProtocolChannelThreadMessage(channel, thr, msg, w.Format, w.Incoming)
}

// SendUserChannelMessage lets a plugin easily send a message directed to
// a specific user in a specific channel without fiddling with the robot
// object. Note that this will fail with UserNotFound if the connector
// can't resolve usernames, or the username isn't mapped to a user ID in
// the protocol's UserMap.
func (r Robot) SendUserChannelMessage(u, ch, msg string, v ...interface{}) robot.RetVal {
	msg, empty := r.prepareMessage("SendUserChannelMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	user := r.tryResolveUser(u)
	channel := r.tryResolveChannel(ch)
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, "", msg, r.Format, r.Incoming)
}

func (w *worker) SendUserChannelMessage(u, ch, msg string, v ...interface{}) robot.RetVal {
	msg, empty := w.prepareMessage("SendUserChannelMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	user := w.tryResolveUser(u)
	channel := w.tryResolveChannel(ch)
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, "", msg, w.Format, w.Incoming)
}

func (r Robot) SendProtocolUserChannelMessage(protocol, u, ch, msg string, v ...interface{}) robot.RetVal {
	msg, empty := r.prepareMessage("SendProtocolUserChannelMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	p := normalizeProtocolName(protocol)
	if p == "" {
		Log(robot.Error, "SendProtocolUserChannelMessage: protocol is required")
		return robot.MissingArguments
	}
	if u == "" && ch == "" {
		Log(robot.Error, "SendProtocolUserChannelMessage: either user or channel is required")
		return robot.MissingArguments
	}
	if _, ok := getRuntimePrimaryProtocol(); ok {
		if p != runtimeConnectorsPrimary() {
			if _, exists := getRuntimeConnector(p); !exists {
				Log(robot.Error, "SendProtocolUserChannelMessage: protocol '%s' is not active", p)
				return robot.Failed
			}
		}
	}
	msgObject := protocolIncoming(r.Incoming, p)
	if u != "" && ch == "" {
		user := r.tryResolveUserForProtocol(p, u)
		return interfaces.SendProtocolUserMessage(user, msg, r.Format, msgObject)
	}
	channel := r.tryResolveChannelForProtocol(p, ch)
	if u == "" {
		return interfaces.SendProtocolChannelThreadMessage(channel, "", msg, r.Format, msgObject)
	}
	user := r.tryResolveUserForProtocol(p, u)
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, "", msg, r.Format, msgObject)
}

func (r Robot) SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) robot.RetVal {
	msg, empty := r.prepareMessage("SendUserChannelThreadMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	user := r.tryResolveUser(u)
	channel := r.tryResolveChannel(ch)
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, thr, msg, r.Format, r.Incoming)
}

func (w *worker) SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) robot.RetVal {
	msg, empty := w.prepareMessage("SendUserChannelThreadMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	user := w.tryResolveUser(u)
	channel := w.tryResolveChannel(ch)
	return interfaces.SendProtocolUserChannelThreadMessage(user, u, channel, thr, msg, w.Format, w.Incoming)
}

// see robot/robot.go
func (r Robot) SendUserMessage(u, msg string, v ...interface{}) robot.RetVal {
	msg, empty := r.prepareMessage("SendUserMessage", msg, v...)
	if empty {
		return robot.Failed
	}
	user := r.tryResolveUser(u)
	return interfaces.SendProtocolUserMessage(user, msg, r.Format, r.Incoming)
}

// see robot/robot.go
func (r Robot) Reply(msg string, v ...interface{}) robot.RetVal {
	msg, empty := r.prepareMessage("Reply", msg, v...)
	if empty {
		return robot.Failed
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

func (w *worker) Reply(msg string, v ...interface{}) robot.RetVal {
	msg, empty := w.prepareMessage("Reply", msg, v...)
	if empty {
		return robot.Failed
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
	msg, empty := r.prepareMessage("ReplyThread", msg, v...)
	if empty {
		return robot.Failed
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
	msg, empty := r.prepareMessage("Say", msg, v...)
	if empty {
		return robot.Failed
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

func (w *worker) Say(msg string, v ...interface{}) robot.RetVal {
	msg, empty := w.prepareMessage("Say", msg, v...)
	if empty {
		return robot.Failed
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
	msg, empty := r.prepareMessage("SayThread", msg, v...)
	if empty {
		return robot.Failed
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
