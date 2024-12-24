package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterPromptingMethods adds the prompt methods to the robot's metatable:
//
//	PromptForReply, PromptThreadForReply, PromptUserForReply, etc.
func (lctx luaContext) RegisterPromptingMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"PromptForReply":                  lctx.robotPromptForReply,
		"PromptThreadForReply":            lctx.robotPromptThreadForReply,
		"PromptUserForReply":              lctx.robotPromptUserForReply,
		"PromptUserChannelForReply":       lctx.robotPromptUserChannelForReply,
		"PromptUserChannelThreadForReply": lctx.robotPromptUserChannelThreadForReply,
	}

	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}

// -------------------------------------------------------------------
// 1. Basic prompting methods
// -------------------------------------------------------------------

// robotPromptForReply prompts for a reply and returns the reply string and return value.
func (lctx luaContext) robotPromptForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2) // required
	prompt := L.CheckString(3)  // required

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("PromptForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptForReply(regexID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// robotPromptThreadForReply prompts for a reply in a threaded context and returns the reply string and return value.
func (lctx luaContext) robotPromptThreadForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("PromptThreadForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptThreadForReply(regexID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// robotPromptUserForReply prompts a specific user for a reply and returns the reply string and return value.
func (lctx luaContext) robotPromptUserForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	prompt := L.CheckString(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("PromptUserForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptUserForReply(regexID, user, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// robotPromptUserChannelForReply prompts a specific user in a specific channel for a reply and returns the reply string and return value.
func (lctx luaContext) robotPromptUserChannelForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	channel := L.CheckString(4)
	prompt := L.CheckString(5)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("PromptUserChannelForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptUserChannelForReply(regexID, user, channel, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// robotPromptUserChannelThreadForReply prompts a specific user in a specific channel and thread for a reply and returns the reply string and return value.
func (lctx luaContext) robotPromptUserChannelThreadForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	channel := L.CheckString(4)
	thread := L.CheckString(5)
	prompt := L.CheckString(6)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("PromptUserChannelThreadForReply")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}
