package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// getRobotWithOptionalFormat is a helper function to:
// 1) Extract the luaRobot from stack index 1
// 2) If present, parse a message format argument
// Returns:
//
//	(robot.Robot, error string) - if errStr != "" then log error
func (lctx luaContext) getRobotWithOptionalFormat(L *glua.LState, caller string, formatIndex int) (robot.Robot, bool) {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.r.Log(robot.Error, fmt.Sprintf("%s called with invalid robot userdata", caller))
		return nil, false
	}

	r := lr.r

	// If the caller supplied an argument for format, parse it
	if L.GetTop() >= formatIndex {
		fmtArg := L.Get(formatIndex)
		if num, isNum := fmtArg.(glua.LNumber); isNum {
			format := robot.MessageFormat(int(num))
			r = r.MessageFormat(format)
		}
	}
	return r, true
}

// robotSay(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:Say("Hello world", fmtRaw)
func (lctx luaContext) robotSay(L *glua.LState) int {
	// 2nd arg is the message string
	msg := L.CheckString(2)

	// Setup Robot + optional format at stack index 3
	r, ok := lctx.getRobotWithOptionalFormat(L, "robotSay", 3)
	if !ok {
		return pushFail(L)
	}

	ret := r.Say(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotReply(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:Reply("Hello user", fmtVariable)
func (lctx luaContext) robotReply(L *glua.LState) int {
	msg := L.CheckString(2)

	r, ok := lctx.getRobotWithOptionalFormat(L, "robotReply", 3)
	if !ok {
		return pushFail(L)
	}

	ret := r.Reply(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSayThread(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SayThread("Hello in a new thread", fmtRaw)
func (lctx luaContext) robotSayThread(L *glua.LState) int {
	msg := L.CheckString(2)

	r, ok := lctx.getRobotWithOptionalFormat(L, "robotSayThread", 3)
	if !ok {
		return pushFail(L)
	}

	ret := r.SayThread(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotReplyThread(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:ReplyThread("Replying in a new thread", fmtFixed)
func (lctx luaContext) robotReplyThread(L *glua.LState) int {
	msg := L.CheckString(2)

	r, ok := lctx.getRobotWithOptionalFormat(L, "robotReplyThread", 3)
	if !ok {
		return pushFail(L)
	}

	ret := r.ReplyThread(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendChannelMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendChannelMessage("my-channel", "Hello channel", fmtFixed)
func (lctx luaContext) robotSendChannelMessage(L *glua.LState) int {
	channel := L.CheckString(2)
	msg := L.CheckString(3)

	// Optional format is 4th arg
	r, ok := lctx.getRobotWithOptionalFormat(L, "robotSendChannelMessage", 4)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendChannelMessage(channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendChannelThreadMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendChannelThreadMessage("my-channel", "thread-id", "Hello thread", fmtRaw)
func (lctx luaContext) robotSendChannelThreadMessage(L *glua.LState) int {
	channel := L.CheckString(2)
	thread := L.CheckString(3)
	msg := L.CheckString(4)

	// Optional format is 5th arg
	r, ok := lctx.getRobotWithOptionalFormat(L, "robotSendChannelThreadMessage", 5)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendChannelThreadMessage(channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendUserMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendUserMessage("some.user", "Hello user", fmtRaw)
func (lctx luaContext) robotSendUserMessage(L *glua.LState) int {
	user := L.CheckString(2)
	msg := L.CheckString(3)

	// Optional format is 4th arg
	r, ok := lctx.getRobotWithOptionalFormat(L, "robotSendUserMessage", 4)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendUserMessage(user, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendUserChannelMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendUserChannelMessage("some.user", "some-channel", "Hello in channel", fmtVariable)
func (lctx luaContext) robotSendUserChannelMessage(L *glua.LState) int {
	user := L.CheckString(2)
	channel := L.CheckString(3)
	msg := L.CheckString(4)

	r, ok := lctx.getRobotWithOptionalFormat(L, "robotSendUserChannelMessage", 5)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendUserChannelMessage(user, channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendUserChannelThreadMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendUserChannelThreadMessage("some.user", "some-channel", "some-thread", "Hello in thread", fmtFixed)
func (lctx luaContext) robotSendUserChannelThreadMessage(L *glua.LState) int {
	user := L.CheckString(2)
	channel := L.CheckString(3)
	thread := L.CheckString(4)
	msg := L.CheckString(5)

	r, ok := lctx.getRobotWithOptionalFormat(L, "robotSendUserChannelThreadMessage", 6)
	if !ok {
		return pushFail(L)
	}

	ret := r.SendUserChannelThreadMessage(user, channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// RegisterMessageMethods can be called from your "registerRobotType" logic
// to add these methods into the robot's metatable.
func (lctx luaContext) RegisterMessageMethods(L *glua.LState) {
	messageMethods := map[string]glua.LGFunction{
		"Say":                          lctx.robotSay,
		"SayThread":                    lctx.robotSayThread,
		"ReplyThread":                  lctx.robotReplyThread,
		"Reply":                        lctx.robotReply,
		"SendChannelMessage":           lctx.robotSendChannelMessage,
		"SendChannelThreadMessage":     lctx.robotSendChannelThreadMessage,
		"SendUserMessage":              lctx.robotSendUserMessage,
		"SendUserChannelMessage":       lctx.robotSendUserChannelMessage,
		"SendUserChannelThreadMessage": lctx.robotSendUserChannelThreadMessage,
	}

	robotIndex := getRobotMethodTable(L)
	// Merge in the new message methods
	L.SetFuncs(robotIndex, messageMethods)
}
