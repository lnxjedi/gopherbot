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
		// Bot/user attribute getters
		"GetBotAttribute":    lctx.robotGetBotAttribute,
		"GetUserAttribute":   lctx.robotGetUserAttribute,
		"GetSenderAttribute": lctx.robotGetSenderAttribute,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
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
		lctx.logBotErr("GetBotAttribute")
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
		lctx.logBotErr("GetUserAttribute")
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
		lctx.logBotErr("GetSenderAttribute")
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.AttributeNotFound))
		return 2
	}

	ret := lr.r.GetSenderAttribute(attribute)
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}
