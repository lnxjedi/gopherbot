package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterAttributeMethods adds the following methods to "robot":
//
//	User(), UserID(), Channel(), ChannelID(), ThreadID(), ThreadedMessage()
//	GetBotAttribute(attr) -> (stringVal, retVal)
//	GetUserAttribute(user, attr) -> (stringVal, retVal)
//	GetSenderAttribute(attr) -> (stringVal, retVal)
func RegisterAttributeMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		// Basic message-based attributes
		"User":            robotUser,
		"UserID":          robotUserID,
		"Channel":         robotChannel,
		"ChannelID":       robotChannelID,
		"ThreadID":        robotThreadID,
		"ThreadedMessage": robotThreadedMessage,

		// Bot/user attribute getters
		"GetBotAttribute":    robotGetBotAttribute,
		"GetUserAttribute":   robotGetUserAttribute,
		"GetSenderAttribute": robotGetSenderAttribute,
	}

	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}

// -------------------------------------------------------------------
// Helper to remove leading/trailing < >
// e.g. "<u0007>" -> "u0007", "<#lua>" -> "#lua"
// -------------------------------------------------------------------
func stripAngleBrackets(s string) string {
	if len(s) >= 2 && s[0] == '<' && s[len(s)-1] == '>' {
		return s[1 : len(s)-1]
	}
	return s
}

// -------------------------------------------------------------------
// 1. Basic message-based attribute getters
// -------------------------------------------------------------------

// robot:User() -> string
// Usage: local u = robot:User()
func robotUser(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		// Log error, return empty
		logBadUserdata(lr, "User")
		L.Push(glua.LString(""))
		return 1
	}
	msg := lr.r.GetMessage()
	if msg == nil {
		// no incoming message
		lr.r.Log(robot.Error, "robot:User() - msg is nil")
		L.Push(glua.LString(""))
		return 1
	}
	L.Push(glua.LString(msg.User))
	return 1
}

// robot:UserID() -> string
func robotUserID(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logBadUserdata(lr, "UserID")
		L.Push(glua.LString(""))
		return 1
	}
	msg := lr.r.GetMessage()
	if msg == nil {
		lr.r.Log(robot.Error, "robot:UserID() - msg is nil")
		L.Push(glua.LString(""))
		return 1
	}
	L.Push(glua.LString(stripAngleBrackets(msg.ProtocolUser)))
	return 1
}

// robot:Channel() -> string
func robotChannel(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logBadUserdata(lr, "Channel")
		L.Push(glua.LString(""))
		return 1
	}
	msg := lr.r.GetMessage()
	if msg == nil {
		lr.r.Log(robot.Error, "robot:Channel() - msg is nil")
		L.Push(glua.LString(""))
		return 1
	}
	L.Push(glua.LString(msg.Channel))
	return 1
}

// robot:ChannelID() -> string
func robotChannelID(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logBadUserdata(lr, "ChannelID")
		L.Push(glua.LString(""))
		return 1
	}
	msg := lr.r.GetMessage()
	if msg == nil {
		lr.r.Log(robot.Error, "robot:ChannelID() - msg is nil")
		L.Push(glua.LString(""))
		return 1
	}
	L.Push(glua.LString(stripAngleBrackets(msg.ProtocolChannel)))
	return 1
}

// robot:ThreadID() -> string
// We fetch it from lr.env["GOPHER_THREAD_ID"] or return ""
func robotThreadID(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logBadUserdata(lr, "ThreadID")
		L.Push(glua.LString(""))
		return 1
	}
	if lr.env == nil {
		lr.r.Log(robot.Error, "robot:ThreadID() - env map is nil")
		L.Push(glua.LString(""))
		return 1
	}
	thrID := lr.env["GOPHER_THREAD_ID"]
	L.Push(glua.LString(thrID))
	return 1
}

// robot:ThreadedMessage() -> bool
// We fetch from lr.env["GOPHER_THREADED_MESSAGE"] == "true"
func robotThreadedMessage(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logBadUserdata(lr, "ThreadedMessage")
		L.Push(glua.LBool(false))
		return 1
	}
	if lr.env == nil {
		lr.r.Log(robot.Error, "robot:ThreadedMessage() - env map is nil")
		L.Push(glua.LBool(false))
		return 1
	}
	thr := (lr.env["GOPHER_THREADED_MESSAGE"] == "true")
	L.Push(glua.LBool(thr))
	return 1
}

// -------------------------------------------------------------------
// 2. Bot/user attribute getters
//    => Returns (attrString, retVal) in Lua
// -------------------------------------------------------------------

// robot:GetBotAttribute(attr) -> (string, intRetVal)
func robotGetBotAttribute(L *glua.LState) int {
	ud := L.CheckUserData(1)
	attribute := L.CheckString(2) // e.g. "name", "alias", etc.

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logBadUserdata(lr, "GetBotAttribute")
		// Return "", retAttributeNotFound or something
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.AttributeNotFound))
		return 2
	}

	ret := lr.r.GetBotAttribute(attribute)
	// ret.Attribute is the string
	// ret.RetVal is e.g. Ok / UserNotFound / ...
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}

// robot:GetUserAttribute(user, attr) -> (string, intRetVal)
func robotGetUserAttribute(L *glua.LState) int {
	ud := L.CheckUserData(1)
	user := L.CheckString(2)
	attribute := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logBadUserdata(lr, "GetUserAttribute")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.AttributeNotFound))
		return 2
	}

	ret := lr.r.GetUserAttribute(user, attribute)
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}

// robot:GetSenderAttribute(attr) -> (string, intRetVal)
func robotGetSenderAttribute(L *glua.LState) int {
	ud := L.CheckUserData(1)
	attribute := L.CheckString(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logBadUserdata(lr, "GetSenderAttribute")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.AttributeNotFound))
		return 2
	}

	ret := lr.r.GetSenderAttribute(attribute)
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}

// Helper for logging "invalid robot userdata" errors
func logBadUserdata(lr *luaRobot, caller string) {
	if lr != nil && lr.r != nil {
		lr.r.Log(robot.Error, fmt.Sprintf("%s called with invalid robot userdata", caller))
	} else {
		// Fallback: no robot to log
		fmt.Printf("[ERR] %s called but robot is nil\n", caller)
	}
}
