// send_messages.go
package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// getBotWithOptionalFormat is a helper function to:
// 1) Extract the luaRobot (ud.Value) from stack index 1
// 2) If present, parse a numeric message format argument (stack index formatIndex)
func (lctx *luaContext) getBotWithOptionalFormat(L *glua.LState, caller string, formatIndex int) (robot.Robot, bool) {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr(caller)
		return nil, false
	}

	r := lr.r

	// If the caller supplied a numeric argument for format, parse and validate it
	if L.GetTop() >= formatIndex {
		fmtArg := L.Get(formatIndex)
		if fmtArg.Type() != glua.LTNumber {
			lr.r.Log(robot.Error, fmt.Sprintf("%s: MessageFormat argument must be a number (Raw=0, Fixed=1, Variable=2)", caller))
			return nil, false
		}

		formatInt := int(fmtArg.(glua.LNumber))
		if !isValidMessageFormat(formatInt) {
			lr.r.Log(robot.Error, fmt.Sprintf("%s: Invalid MessageFormat value: %d. Must be Raw=0, Fixed=1, or Variable=2", caller, formatInt))
			return nil, false
		}

		r = r.MessageFormat(robot.MessageFormat(formatInt))
	}

	return r, true
}

// botSay(luaState) -> retVal
// Usage in Lua: local ret = bot:Say("Hello world", fmtRaw)
func (lctx luaContext) botSay(L *glua.LState) int {
	msg := L.CheckString(2)
	r, ok := lctx.getBotWithOptionalFormat(L, "botSay", 3)
	if !ok {
		return pushFail(L)
	}

	ret := r.Say(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSayThread(luaState) -> retVal
// Usage in Lua: local ret = bot:SayThread("Hello in a new thread", fmtRaw)
func (lctx luaContext) botSayThread(L *glua.LState) int {
	msg := L.CheckString(2)
	r, ok := lctx.getBotWithOptionalFormat(L, "botSayThread", 3)
	if !ok {
		return pushFail(L)
	}

	ret := r.SayThread(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botReply(luaState) -> retVal
// Usage: local ret = bot:Reply("Hello user", fmtVariable)
func (lctx luaContext) botReply(L *glua.LState) int {
	msg := L.CheckString(2)
	r, ok := lctx.getBotWithOptionalFormat(L, "botReply", 3)
	if !ok {
		return pushFail(L)
	}

	ret := r.Reply(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botReplyThread(luaState) -> retVal
// Usage: local ret = bot:ReplyThread("Replying in a new thread", fmtFixed)
func (lctx luaContext) botReplyThread(L *glua.LState) int {
	msg := L.CheckString(2)
	r, ok := lctx.getBotWithOptionalFormat(L, "botReplyThread", 3)
	if !ok {
		return pushFail(L)
	}

	ret := r.ReplyThread(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendChannelMessage(luaState) -> retVal
// Usage: local ret = bot:SendChannelMessage("my-channel", "Hello channel", fmtFixed)
func (lctx luaContext) botSendChannelMessage(L *glua.LState) int {
	channel := L.CheckString(2)
	msg := L.CheckString(3)
	r, ok := lctx.getBotWithOptionalFormat(L, "botSendChannelMessage", 4)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendChannelMessage(channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendChannelThreadMessage(luaState) -> retVal
// Usage: local ret = bot:SendChannelThreadMessage("my-channel", "thread-id", "Hello thread", fmtRaw)
func (lctx luaContext) botSendChannelThreadMessage(L *glua.LState) int {
	channel := L.CheckString(2)
	thread := L.CheckString(3)
	msg := L.CheckString(4)
	r, ok := lctx.getBotWithOptionalFormat(L, "botSendChannelThreadMessage", 5)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendChannelThreadMessage(channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserMessage("some.user", "Hello user", fmtRaw)
func (lctx luaContext) botSendUserMessage(L *glua.LState) int {
	user := L.CheckString(2)
	msg := L.CheckString(3)
	r, ok := lctx.getBotWithOptionalFormat(L, "botSendUserMessage", 4)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendUserMessage(user, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserChannelMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserChannelMessage("some.user", "some-channel", "Hello in channel", fmtVariable)
func (lctx luaContext) botSendUserChannelMessage(L *glua.LState) int {
	user := L.CheckString(2)
	channel := L.CheckString(3)
	msg := L.CheckString(4)
	r, ok := lctx.getBotWithOptionalFormat(L, "botSendUserChannelMessage", 5)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendUserChannelMessage(user, channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendUserChannelThreadMessage(luaState) -> retVal
// Usage: local ret = bot:SendUserChannelThreadMessage("some.user", "some-channel", "some-thread", "Hello in thread", fmtFixed)
func (lctx luaContext) botSendUserChannelThreadMessage(L *glua.LState) int {
	user := L.CheckString(2)
	channel := L.CheckString(3)
	thread := L.CheckString(4)
	msg := L.CheckString(5)
	r, ok := lctx.getBotWithOptionalFormat(L, "botSendUserChannelThreadMessage", 6)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendUserChannelThreadMessage(user, channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// RegisterMessageMethods merges message-related methods into the "bot" metatable
func (lctx luaContext) RegisterMessageMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"Say":                          lctx.botSay,
		"SayThread":                    lctx.botSayThread,
		"Reply":                        lctx.botReply,
		"ReplyThread":                  lctx.botReplyThread,
		"SendChannelMessage":           lctx.botSendChannelMessage,
		"SendChannelThreadMessage":     lctx.botSendChannelThreadMessage,
		"SendUserMessage":              lctx.botSendUserMessage,
		"SendUserChannelMessage":       lctx.botSendUserChannelMessage,
		"SendUserChannelThreadMessage": lctx.botSendUserChannelThreadMessage,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}
