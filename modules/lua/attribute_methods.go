package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterAttributeMethods adds the following methods to "robot":
//
//	User(), UserID(), Channel(), ChannelID(), ThreadID(), ThreadedMessage()
//	GetBotAttribute(attr) -> (stringVal, retVal)
//	GetUserAttribute(user, attr) -> (stringVal, retVal)
//	GetSenderAttribute(attr) -> (stringVal, retVal)
func (lctx luaContext) RegisterAttributeMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		// Basic message-based attributes
		"User":            lctx.robotUser,
		"UserID":          lctx.robotUserID,
		"Channel":         lctx.robotChannel,
		"ChannelID":       lctx.robotChannelID,
		"ThreadID":        lctx.robotThreadID,
		"ThreadedMessage": lctx.robotThreadedMessage,

		// Bot/user attribute getters
		"GetBotAttribute":    lctx.robotGetBotAttribute,
		"GetUserAttribute":   lctx.robotGetUserAttribute,
		"GetSenderAttribute": lctx.robotGetSenderAttribute,
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

// robotUser returns the username associated with the current message.
// Usage: local u = robot:User()
func (lctx luaContext) robotUser(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("User")
		L.Push(glua.LString(""))
		return 1
	}

	msg := lr.r.GetMessage()
	if msg == nil {
		// No incoming message
		lr.r.Log(robot.Error, "robot:User() - msg is nil")
		L.Push(glua.LString(""))
		return 1
	}

	L.Push(glua.LString(msg.User))
	return 1
}

// robotUserID returns the user ID associated with the current message.
// Usage: local uid = robot:UserID()
func (lctx luaContext) robotUserID(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("UserID")
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

// robotChannel returns the channel name associated with the current message.
// Usage: local channel = robot:Channel()
func (lctx luaContext) robotChannel(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("Channel")
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

// robotChannelID returns the channel ID associated with the current message.
// Usage: local channelID = robot:ChannelID()
func (lctx luaContext) robotChannelID(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("ChannelID")
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

// robotThreadID returns the thread ID from the environment.
// Usage: local threadID = robot:ThreadID()
func (lctx luaContext) robotThreadID(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("ThreadID")
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

// robotThreadedMessage returns whether the message is threaded.
// Usage: local isThreaded = robot:ThreadedMessage()
func (lctx luaContext) robotThreadedMessage(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("ThreadedMessage")
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

// robotGetBotAttribute retrieves a bot attribute.
// Usage: local attr, ret = robot:GetBotAttribute("name")
func (lctx luaContext) robotGetBotAttribute(L *glua.LState) int {
	ud := L.CheckUserData(1)
	attribute := L.CheckString(2) // e.g., "name", "alias", etc.

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("GetBotAttribute")
		// Return "", robot.AttributeNotFound
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.AttributeNotFound))
		return 2
	}

	ret := lr.r.GetBotAttribute(attribute)
	// ret.Attribute is the string
	// ret.RetVal is e.g., Ok / UserNotFound / ...
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}

// robotGetUserAttribute retrieves a user attribute.
// Usage: local attr, ret = robot:GetUserAttribute("user123", "role")
func (lctx luaContext) robotGetUserAttribute(L *glua.LState) int {
	ud := L.CheckUserData(1)
	user := L.CheckString(2)
	attribute := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("GetUserAttribute")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.AttributeNotFound))
		return 2
	}

	ret := lr.r.GetUserAttribute(user, attribute)
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}

// robotGetSenderAttribute retrieves an attribute of the message sender.
// Usage: local attr, ret = robot:GetSenderAttribute("status")
func (lctx luaContext) robotGetSenderAttribute(L *glua.LState) int {
	ud := L.CheckUserData(1)
	attribute := L.CheckString(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("GetSenderAttribute")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.AttributeNotFound))
		return 2
	}

	ret := lr.r.GetSenderAttribute(attribute)
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}
