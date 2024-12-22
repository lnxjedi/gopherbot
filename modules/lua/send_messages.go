package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// getRobotWithOptionalFormat is a helper function to:
// 1) Extract the luaRobot from stack index 1
// 2) If present, parse a message format argument
// Returns:
//
//	(robot.Robot, error string) - if errStr != "" then raise Lua error
func getRobotWithOptionalFormat(L *glua.LState, formatIndex int) (robot.Robot, string) {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		return nil, "invalid robot userdata"
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
	return r, ""
}

// robotSay(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:Say("Hello world", fmtRaw)
func robotSay(L *glua.LState) int {
	// 2nd arg is the message string
	msg := L.CheckString(2)

	// Setup Robot + optional format at stack index 3
	r, errStr := getRobotWithOptionalFormat(L, 3)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.Say(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotReply(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:Reply("Hello user", fmtVariable)
func robotReply(L *glua.LState) int {
	msg := L.CheckString(2)

	r, errStr := getRobotWithOptionalFormat(L, 3)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.Reply(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSayThread(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SayThread("Hello in a new thread", fmtRaw)
func robotSayThread(L *glua.LState) int {
	msg := L.CheckString(2)

	r, errStr := getRobotWithOptionalFormat(L, 3)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.SayThread(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotReplyThread(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:ReplyThread("Replying in a new thread", fmtFixed)
func robotReplyThread(L *glua.LState) int {
	msg := L.CheckString(2)

	r, errStr := getRobotWithOptionalFormat(L, 3)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.ReplyThread(msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendChannelMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendChannelMessage("my-channel", "Hello channel", fmtFixed)
func robotSendChannelMessage(L *glua.LState) int {
	channel := L.CheckString(2)
	msg := L.CheckString(3)

	// Optional format is 4th arg
	r, errStr := getRobotWithOptionalFormat(L, 4)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.SendChannelMessage(channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendChannelThreadMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendChannelThreadMessage("my-channel", "thread-id", "Hello thread", fmtRaw)
func robotSendChannelThreadMessage(L *glua.LState) int {
	channel := L.CheckString(2)
	thread := L.CheckString(3)
	msg := L.CheckString(4)

	// Optional format is 5th arg
	r, errStr := getRobotWithOptionalFormat(L, 5)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.SendChannelThreadMessage(channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendUserMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendUserMessage("some.user", "Hello user", fmtRaw)
func robotSendUserMessage(L *glua.LState) int {
	user := L.CheckString(2)
	msg := L.CheckString(3)

	r, errStr := getRobotWithOptionalFormat(L, 4)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.SendUserMessage(user, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendUserChannelMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendUserChannelMessage("some.user", "some-channel", "Hello in channel", fmtVariable)
func robotSendUserChannelMessage(L *glua.LState) int {
	user := L.CheckString(2)
	channel := L.CheckString(3)
	msg := L.CheckString(4)

	r, errStr := getRobotWithOptionalFormat(L, 5)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.SendUserChannelMessage(user, channel, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// robotSendUserChannelThreadMessage(luaState) -> retVal
// Usage in Lua:
//
//	local ret = robot:SendUserChannelThreadMessage("some.user", "some-channel", "some-thread", "Hello in thread", fmtFixed)
func robotSendUserChannelThreadMessage(L *glua.LState) int {
	user := L.CheckString(2)
	channel := L.CheckString(3)
	thread := L.CheckString(4)
	msg := L.CheckString(5)

	r, errStr := getRobotWithOptionalFormat(L, 6)
	if errStr != "" {
		L.RaiseError(errStr)
		return 0
	}

	ret := r.SendUserChannelThreadMessage(user, channel, thread, msg)
	L.Push(glua.LNumber(ret))
	return 1
}

// RegisterMessageMethods can be called from your "registerRobotType" logic
// to add these methods into the robot's metatable.
func RegisterMessageMethods(L *glua.LState) {
	messageMethods := map[string]glua.LGFunction{
		"Say":                          robotSay,
		"SayThread":                    robotSayThread,
		"ReplyThread":                  robotReplyThread,
		"Reply":                        robotReply,
		"SendChannelMessage":           robotSendChannelMessage,
		"SendChannelThreadMessage":     robotSendChannelThreadMessage,
		"SendUserMessage":              robotSendUserMessage,
		"SendUserChannelMessage":       robotSendUserChannelMessage,
		"SendUserChannelThreadMessage": robotSendUserChannelThreadMessage,
	}

	robotIndex := getRobotMethodTable(L)
	// Merge in the new message methods
	L.SetFuncs(robotIndex, messageMethods)
}
