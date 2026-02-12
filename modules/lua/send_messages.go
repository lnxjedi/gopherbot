package lua

import (
	glua "github.com/yuin/gopher-lua"
)

// botSendChannelMessage(luaState) -> retVal
// Usage: local ret = bot:SendChannelMessage("my-channel", "Hello channel", fmtFixed)
func (lctx *luaContext) botSendChannelMessage(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "SendChannelMessage", 4)

	channel := L.CheckString(2)
	msg := L.CheckString(3)

	// Hard fail on empty channel
	if channel == "" {
		L.RaiseError("SendChannelMessage: channel name must not be empty")
		return 0
	}
	// If msg is empty, the engine will return robot.Fail or log a warning.

	ret := r.SendChannelMessage(channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendChannelThreadMessage(luaState) -> retVal
// Usage: local ret = bot:SendChannelThreadMessage("my-channel", "thread-id", "Hello thread", fmtRaw)
func (lctx *luaContext) botSendChannelThreadMessage(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "SendChannelThreadMessage", 5)

	channel := L.CheckString(2)
	thread := L.CheckString(3)
	msg := L.CheckString(4)

	if channel == "" {
		L.RaiseError("SendChannelThreadMessage: channel name must not be empty")
		return 0
	}
	// thread can be empty => top-level or no thread
	// message can be empty => engine handles it

	ret := r.SendChannelThreadMessage(channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserMessage("some.user", "Hello user", fmtRaw)
func (lctx *luaContext) botSendUserMessage(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "SendUserMessage", 4)

	user := L.CheckString(2)
	msg := L.CheckString(3)

	if user == "" {
		L.RaiseError("SendUserMessage: user argument must not be empty")
		return 0
	}
	// Allow empty msg => engine returns robot.Fail / logs warning

	ret := r.SendUserMessage(user, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserChannelMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserChannelMessage("some.user", "some-channel", "Hello in channel", fmtVariable)
func (lctx *luaContext) botSendUserChannelMessage(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "SendUserChannelMessage", 5)

	user := L.CheckString(2)
	channel := L.CheckString(3)
	msg := L.CheckString(4)

	if user == "" {
		L.RaiseError("SendUserChannelMessage: user must not be empty")
		return 0
	}
	if channel == "" {
		L.RaiseError("SendUserChannelMessage: channel must not be empty")
		return 0
	}

	ret := r.SendUserChannelMessage(user, channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendProtocolUserChannelMessage(luaState) -> retVal
// Usage: local ret = bot:SendProtocolUserChannelMessage("ssh", "some.user", "some-channel", "Hello", fmtRaw)
func (lctx *luaContext) botSendProtocolUserChannelMessage(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "SendProtocolUserChannelMessage", 6)

	protocol := L.CheckString(2)
	user := L.CheckString(3)
	channel := L.CheckString(4)
	msg := L.CheckString(5)

	if protocol == "" {
		L.RaiseError("SendProtocolUserChannelMessage: protocol must not be empty")
		return 0
	}

	ret := r.SendProtocolUserChannelMessage(protocol, user, channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserChannelThreadMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserChannelThreadMessage("some.user", "some-channel", "some-thread", "Hello", fmtFixed)
func (lctx *luaContext) botSendUserChannelThreadMessage(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "SendUserChannelThreadMessage", 6)

	user := L.CheckString(2)
	channel := L.CheckString(3)
	thread := L.CheckString(4)
	msg := L.CheckString(5)

	if user == "" {
		L.RaiseError("SendUserChannelThreadMessage: user must not be empty")
		return 0
	}
	if channel == "" {
		L.RaiseError("SendUserChannelThreadMessage: channel must not be empty")
		return 0
	}
	// thread can be empty
	// message can be empty => engine returns fail / logs warning

	ret := r.SendUserChannelThreadMessage(user, channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSay(luaState) -> retVal
// Usage: local ret = bot:Say("some text", fmtRaw)
func (lctx *luaContext) botSay(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "Say", 3)
	msg := L.CheckString(2)
	// Let the engine handle empty messages => returns robot.Fail
	ret := r.Say(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSayThread(luaState) -> retVal
// Usage: local ret = bot:SayThread("some text", fmtFixed)
func (lctx *luaContext) botSayThread(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "SayThread", 3)
	msg := L.CheckString(2)
	ret := r.SayThread(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botReply(luaState) -> retVal
// Usage: local ret = bot:Reply("some text", fmtVariable)
func (lctx *luaContext) botReply(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "Reply", 3)
	msg := L.CheckString(2)
	ret := r.Reply(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botReplyThread(luaState) -> retVal
// Usage: local ret = bot:ReplyThread("some text", fmtRaw)
func (lctx *luaContext) botReplyThread(L *glua.LState) int {
	r := lctx.getOptionalFormattedRobot(L, "ReplyThread", 3)
	msg := L.CheckString(2)
	ret := r.ReplyThread(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// RegisterMessageMethods merges message-related methods into the "bot" metatable
func (lctx *luaContext) RegisterMessageMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		// Existing “Send*” methods
		"SendChannelMessage":             lctx.botSendChannelMessage,
		"SendChannelThreadMessage":       lctx.botSendChannelThreadMessage,
		"SendUserMessage":                lctx.botSendUserMessage,
		"SendUserChannelMessage":         lctx.botSendUserChannelMessage,
		"SendProtocolUserChannelMessage": lctx.botSendProtocolUserChannelMessage,
		"SendUserChannelThreadMessage":   lctx.botSendUserChannelThreadMessage,
		"Say":                            lctx.botSay,
		"SayThread":                      lctx.botSayThread,
		"Reply":                          lctx.botReply,
		"ReplyThread":                    lctx.botReplyThread,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}
