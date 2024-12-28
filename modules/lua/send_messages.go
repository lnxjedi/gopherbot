// send_messages.go
package lua

import (
	glua "github.com/yuin/gopher-lua"
)

// botSendChannelMessage(luaState) -> retVal
// Usage: local ret = bot:SendChannelMessage("my-channel", "Hello channel", fmtFixed)
func (lctx luaContext) botSendChannelMessage(L *glua.LState) int {
	r, ok := lctx.getOptionalFormattedRobot(L, "SendChannelMessage", 4)
	if !ok {
		return pushFail(L)
	}
	channel := L.CheckString(2)
	msg := L.CheckString(3)

	ret := r.SendChannelMessage(channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendChannelThreadMessage(luaState) -> retVal
// Usage: local ret = bot:SendChannelThreadMessage("my-channel", "thread-id", "Hello thread", fmtRaw)
func (lctx luaContext) botSendChannelThreadMessage(L *glua.LState) int {
	r, ok := lctx.getOptionalFormattedRobot(L, "SendChannelThreadMessage", 5)
	if !ok {
		return pushFail(L)
	}
	channel := L.CheckString(2)
	thread := L.CheckString(3)
	msg := L.CheckString(4)

	ret := r.SendChannelThreadMessage(channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserMessage("some.user", "Hello user", fmtRaw)
func (lctx luaContext) botSendUserMessage(L *glua.LState) int {
	r, ok := lctx.getOptionalFormattedRobot(L, "SendUserMessage", 4)
	if !ok {
		return pushFail(L)
	}
	user := L.CheckString(2)
	msg := L.CheckString(3)

	ret := r.SendUserMessage(user, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserChannelMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserChannelMessage("some.user", "some-channel", "Hello in channel", fmtVariable)
func (lctx luaContext) botSendUserChannelMessage(L *glua.LState) int {
	r, ok := lctx.getOptionalFormattedRobot(L, "SendUserChannelMessage", 5)
	if !ok {
		return pushFail(L)
	}
	user := L.CheckString(2)
	channel := L.CheckString(3)
	msg := L.CheckString(4)

	ret := r.SendUserChannelMessage(user, channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserChannelThreadMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserChannelThreadMessage("some.user", "some-channel", "some-thread", "Hello in thread", fmtFixed)
func (lctx luaContext) botSendUserChannelThreadMessage(L *glua.LState) int {
	r, ok := lctx.getOptionalFormattedRobot(L, "SendUserChannelThreadMessage", 6)
	if !ok {
		return pushFail(L)
	}
	user := L.CheckString(2)
	channel := L.CheckString(3)
	thread := L.CheckString(4)
	msg := L.CheckString(5)

	ret := r.SendUserChannelThreadMessage(user, channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// RegisterMessageMethods merges message-related methods into the "bot" metatable
func (lctx luaContext) RegisterMessageMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"SendChannelMessage":           lctx.botSendChannelMessage,
		"SendChannelThreadMessage":     lctx.botSendChannelThreadMessage,
		"SendUserMessage":              lctx.botSendUserMessage,
		"SendUserChannelMessage":       lctx.botSendUserChannelMessage,
		"SendUserChannelThreadMessage": lctx.botSendUserChannelThreadMessage,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}
