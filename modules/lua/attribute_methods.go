package lua

import (
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
		"GetBotAttribute":    lctx.botGetBotAttribute,
		"GetUserAttribute":   lctx.botGetUserAttribute,
		"GetSenderAttribute": lctx.botGetSenderAttribute,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// -------------------------------------------------------------------
// 2. Bot/user attribute getters
//    => Returns (attrString, retVal) in Lua
// -------------------------------------------------------------------

// botGetBotAttribute retrieves a bot attribute.
// Usage: local attr, ret = robot:GetBotAttribute("name")
func (lctx luaContext) botGetBotAttribute(L *glua.LState) int {
	r := lctx.getRobot(L, "GetBotAttribute")
	attribute := L.CheckString(2) // e.g., "name", "alias", etc.

	if attribute == "" {
		L.RaiseError("GetBotAttribute: attribute must not be empty")
	}

	ret := r.GetBotAttribute(attribute)
	// ret.Attribute is the string
	// ret.RetVal is e.g., Ok / UserNotFound / ...
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}

// botGetUserAttribute retrieves a user attribute.
// Usage: local attr, ret = robot:GetUserAttribute("user123", "role")
func (lctx luaContext) botGetUserAttribute(L *glua.LState) int {
	r := lctx.getRobot(L, "GetUserAttribute")
	user := L.CheckString(2)
	attribute := L.CheckString(3)

	if user == "" {
		L.RaiseError("GetUserAttribute: user must not be empty")
	}
	if attribute == "" {
		L.RaiseError("GetUserAttribute: attribute must not be empty")
	}

	ret := r.GetUserAttribute(user, attribute)
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}

// botGetSenderAttribute retrieves an attribute of the message sender.
// Usage: local attr, ret = robot:GetSenderAttribute("status")
func (lctx luaContext) botGetSenderAttribute(L *glua.LState) int {
	r := lctx.getRobot(L, "GetSenderAttribute")
	attribute := L.CheckString(2)

	if attribute == "" {
		L.RaiseError("GetSenderAttribute: attribute must not be empty")
	}

	ret := r.GetSenderAttribute(attribute)
	L.Push(glua.LString(ret.Attribute))
	L.Push(glua.LNumber(ret.RetVal))
	return 2
}
