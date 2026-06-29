package bot

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

func queueShutdownFarewell(r Robot, message string) {
	farewell := newShutdownFarewell(r, message)
	if farewell == nil {
		return
	}
	state.Lock()
	state.pendingShutdownFarewell = farewell
	state.Unlock()
}

func newShutdownFarewell(r Robot, message string) *shutdownFarewellMessage {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	incoming := (*robot.ConnectorMessage)(nil)
	if r.Incoming != nil {
		incomingCopy := *r.Incoming
		incoming = &incomingCopy
	}
	user := r.ProtocolUser
	if user == "" {
		user = r.User
	}
	channel := r.ProtocolChannel
	if channel == "" {
		channel = r.Channel
	}
	farewell := &shutdownFarewellMessage{
		user:     user,
		username: r.User,
		channel:  channel,
		message:  message,
		format:   r.Format,
		incoming: incoming,
		direct:   r.Channel == "",
	}
	if incoming != nil && incoming.ThreadedMessage {
		farewell.thread = incoming.ThreadID
	}
	if r.tid != 0 {
		w := getLockedWorker(r.tid)
		farewell.botUser = w.BotUser
		w.Unlock()
	}
	return farewell
}

func sendPendingShutdownFarewell() bool {
	state.Lock()
	farewell := state.pendingShutdownFarewell
	state.pendingShutdownFarewell = nil
	state.Unlock()

	if farewell == nil {
		return false
	}
	if interfaces.Connector == nil {
		Log(robot.Error, "Shutdown farewell queued but connector runtime is unavailable")
		return false
	}
	var ret robot.RetVal
	if farewell.direct {
		ret = interfaces.SendProtocolUserMessage(farewell.user, farewell.message, farewell.format, farewell.incoming)
	} else if farewell.botUser {
		ret = interfaces.SendProtocolChannelThreadMessage(farewell.channel, farewell.thread, farewell.username+": "+farewell.message, farewell.format, farewell.incoming)
	} else {
		ret = interfaces.SendProtocolUserChannelThreadMessage(farewell.user, farewell.username, farewell.channel, farewell.thread, farewell.message, farewell.format, farewell.incoming)
	}
	if ret != robot.Ok {
		Log(robot.Error, "Sending shutdown farewell failed: %s", ret)
		return false
	}
	return true
}
