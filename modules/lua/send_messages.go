// send_messages.go
package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// getBotWithOptionalFormat is a helper function to:
// 1) Extract the luaRobot (ud.Value) from stack index 1
// 2) If present, parse a numeric message format argument (stack index formatIndex)
func (lctx luaContext) getBotWithOptionalFormat(L *glua.LState, caller string, formatIndex int) (robot.Robot, *luaRobot, bool) {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr(caller)
		return nil, nil, false
	}

	r := lr.r

	// If the caller supplied a numeric argument for format, parse it
	if L.GetTop() >= formatIndex {
		fmtArg := L.Get(formatIndex)
		if num, isNum := fmtArg.(glua.LNumber); isNum {
			format := robot.MessageFormat(int(num))
			r = r.MessageFormat(format)
		}
	}
	return r, lr, true
}

// botSay(luaState) -> retVal
// Usage in Lua: local ret = bot:Say("Hello world", fmtRaw)
func (lctx luaContext) botSay(L *glua.LState) int {
	msg := L.CheckString(2)
	r, lr, ok := lctx.getBotWithOptionalFormat(L, "botSay", 3)
	if !ok {
		return pushFail(L)
	}

	// Extract fields from *this* bot
	user, _ := lr.fields["user"].(string)
	channel, _ := lr.fields["channel"].(string)
	threadedMessage, _ := lr.fields["threaded_message"].(bool)
	threadID, _ := lr.fields["thread_id"].(string)

	if channel == "" {
		// Send to user; if user = "" it'll return an error code
		ret := r.SendUserMessage(user, msg)
		L.Push(glua.LNumber(ret))
		return 1
	}

	// If channel != "", possibly thread
	thread := ""
	if threadedMessage {
		thread = threadID
	}
	ret := r.SendChannelThreadMessage(channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSayThread(luaState) -> retVal
// Usage in Lua: local ret = bot:SayThread("Hello in a new thread", fmtRaw)
func (lctx luaContext) botSayThread(L *glua.LState) int {
	msg := L.CheckString(2)
	r, lr, ok := lctx.getBotWithOptionalFormat(L, "botSayThread", 3)
	if !ok {
		return pushFail(L)
	}

	user, _ := lr.fields["user"].(string)
	channel, _ := lr.fields["channel"].(string)
	threadID, _ := lr.fields["thread_id"].(string)

	if channel == "" {
		ret := r.SendUserMessage(user, msg)
		L.Push(glua.LNumber(ret))
		return 1
	}

	ret := r.SendChannelThreadMessage(channel, threadID, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botReply(luaState) -> retVal
// Usage: local ret = bot:Reply("Hello user", fmtVariable)
func (lctx luaContext) botReply(L *glua.LState) int {
	msg := L.CheckString(2)
	r, lr, ok := lctx.getBotWithOptionalFormat(L, "botReply", 3)
	if !ok {
		return pushFail(L)
	}

	user, _ := lr.fields["user"].(string)
	channel, _ := lr.fields["channel"].(string)
	threadedMessage, _ := lr.fields["threaded_message"].(bool)
	threadID, _ := lr.fields["thread_id"].(string)

	if channel == "" {
		ret := r.SendUserMessage(user, msg)
		L.Push(glua.LNumber(ret))
		return 1
	}

	thread := ""
	if threadedMessage {
		thread = threadID
	}
	ret := r.SendUserChannelThreadMessage(user, channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botReplyThread(luaState) -> retVal
// Usage: local ret = bot:ReplyThread("Replying in a new thread", fmtFixed)
func (lctx luaContext) botReplyThread(L *glua.LState) int {
	msg := L.CheckString(2)
	r, lr, ok := lctx.getBotWithOptionalFormat(L, "botReplyThread", 3)
	if !ok {
		return pushFail(L)
	}

	user, _ := lr.fields["user"].(string)
	channel, _ := lr.fields["channel"].(string)
	threadID, _ := lr.fields["thread_id"].(string)

	if channel == "" {
		ret := r.SendUserMessage(user, msg)
		L.Push(glua.LNumber(ret))
		return 1
	}

	ret := r.SendUserChannelThreadMessage(user, channel, threadID, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// botSendChannelMessage(luaState) -> retVal
// Usage: local ret = bot:SendChannelMessage("my-channel", "Hello channel", fmtFixed)
func (lctx luaContext) botSendChannelMessage(L *glua.LState) int {
	channel := L.CheckString(2)
	msg := L.CheckString(3)
	r, _, ok := lctx.getBotWithOptionalFormat(L, "botSendChannelMessage", 4)
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
	r, _, ok := lctx.getBotWithOptionalFormat(L, "botSendChannelThreadMessage", 5)
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
	r, _, ok := lctx.getBotWithOptionalFormat(L, "botSendUserMessage", 4)
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
	r, _, ok := lctx.getBotWithOptionalFormat(L, "botSendUserChannelMessage", 5)
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
	r, _, ok := lctx.getBotWithOptionalFormat(L, "botSendUserChannelThreadMessage", 6)
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
