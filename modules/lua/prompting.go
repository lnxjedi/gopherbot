package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterPromptingMethods adds the prompt methods to the robot's metatable:
//
//	PromptForReply, PromptThreadForReply, PromptUserForReply, etc.
func RegisterPromptingMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"PromptForReply":                  robotPromptForReply,
		"PromptThreadForReply":            robotPromptThreadForReply,
		"PromptUserForReply":              robotPromptUserForReply,
		"PromptUserChannelForReply":       robotPromptUserChannelForReply,
		"PromptUserChannelThreadForReply": robotPromptUserChannelThreadForReply,
	}

	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}

// -------------------------------------------------------------------
// 1) robot:PromptForReply(regexID, prompt) -> (replyString, retVal)
// -------------------------------------------------------------------
func robotPromptForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2) // required
	prompt := L.CheckString(3)  // required

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		// Log an error, return ("", some error code)
		logErr(lr, "PromptForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptForReply(regexID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// -------------------------------------------------------------------
// 2) robot:PromptThreadForReply(regexID, prompt) -> (replyString, retVal)
// -------------------------------------------------------------------
func robotPromptThreadForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "PromptThreadForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptThreadForReply(regexID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// -------------------------------------------------------------------
// 3) robot:PromptUserForReply(regexID, user, prompt) -> (replyString, retVal)
// -------------------------------------------------------------------
func robotPromptUserForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	prompt := L.CheckString(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "PromptUserForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptUserForReply(regexID, user, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// -------------------------------------------------------------------
// 4) robot:PromptUserChannelForReply(regexID, user, channel, prompt) -> (replyString, retVal)
// -------------------------------------------------------------------
func robotPromptUserChannelForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	channel := L.CheckString(4)
	prompt := L.CheckString(5)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "PromptUserChannelForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptUserChannelForReply(regexID, user, channel, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// -------------------------------------------------------------------
// 5) robot:PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt) -> (replyString, retVal)
// -------------------------------------------------------------------
func robotPromptUserChannelThreadForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	channel := L.CheckString(4)
	thread := L.CheckString(5)
	prompt := L.CheckString(6)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "PromptUserChannelThreadForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}
