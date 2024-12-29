package lua

import (
	glua "github.com/yuin/gopher-lua"
)

// RegisterPromptingMethods attaches all PromptForReply methods to the "bot" metatable
func (lctx luaContext) RegisterPromptingMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"PromptForReply":                  lctx.botPromptForReply,
		"PromptThreadForReply":            lctx.botPromptThreadForReply,
		"PromptUserForReply":              lctx.botPromptUserForReply,
		"PromptUserChannelForReply":       lctx.botPromptUserChannelForReply,
		"PromptUserChannelThreadForReply": lctx.botPromptUserChannelThreadForReply,
	}
	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// -------------------------------------------------------------------
// 1) botPromptForReply(luaState)
// Usage in Lua: local reply, retVal = bot:PromptForReply("someRegexID", "Please reply")
// -------------------------------------------------------------------
func (lctx luaContext) botPromptForReply(L *glua.LState) int {
	r := lctx.getRobot(L, "PromptForReply")
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	// regexID should not be empty
	if regexID == "" {
		L.RaiseError("PromptForReply: regexID must not be empty")
		return 0
	}
	// prompt can be empty => let the engine handle it

	reply, ret := r.PromptForReply(regexID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// -------------------------------------------------------------------
// 2) botPromptThreadForReply(luaState)
// Usage: local reply, retVal = bot:PromptThreadForReply("someRegexID", "Please reply in thread")
// -------------------------------------------------------------------
func (lctx luaContext) botPromptThreadForReply(L *glua.LState) int {
	r := lctx.getRobot(L, "PromptThreadForReply")
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	if regexID == "" {
		L.RaiseError("PromptThreadForReply: regexID must not be empty")
		return 0
	}

	reply, ret := r.PromptThreadForReply(regexID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// -------------------------------------------------------------------
// 3) botPromptUserForReply(luaState)
// Usage: local reply, retVal = bot:PromptUserForReply("someRegexID", "someUser", "Hello user")
// -------------------------------------------------------------------
func (lctx luaContext) botPromptUserForReply(L *glua.LState) int {
	r := lctx.getRobot(L, "PromptUserForReply")
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	prompt := L.CheckString(4)

	if regexID == "" {
		L.RaiseError("PromptUserForReply: regexID must not be empty")
		return 0
	}
	if user == "" {
		L.RaiseError("PromptUserForReply: user must not be empty")
		return 0
	}
	// prompt can be empty => let engine handle

	reply, ret := r.PromptUserForReply(regexID, user, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// -------------------------------------------------------------------
// 4) botPromptUserChannelForReply(luaState)
// Usage: local reply, retVal = bot:PromptUserChannelForReply("someRegexID", "someUser", "someChannel", "Prompt text")
// -------------------------------------------------------------------
func (lctx luaContext) botPromptUserChannelForReply(L *glua.LState) int {
	r := lctx.getRobot(L, "PromptUserChannelForReply")
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	channel := L.CheckString(4)
	prompt := L.CheckString(5)

	if regexID == "" {
		L.RaiseError("PromptUserChannelForReply: regexID must not be empty")
		return 0
	}
	if user == "" {
		L.RaiseError("PromptUserChannelForReply: user must not be empty")
		return 0
	}
	if channel == "" {
		L.RaiseError("PromptUserChannelForReply: channel must not be empty")
		return 0
	}

	reply, ret := r.PromptUserChannelForReply(regexID, user, channel, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// -------------------------------------------------------------------
// 5) botPromptUserChannelThreadForReply(luaState)
// Usage: local reply, retVal = bot:PromptUserChannelThreadForReply("someRegexID", "someUser", "someChannel", "someThread", "Prompt text")
// Note: The underlying Go code as shown uses the "thread" param but actually passes
//
//	an empty string to promptInternal. If you later fix that in Go, it'll use "thread" properly.
//
// -------------------------------------------------------------------
func (lctx luaContext) botPromptUserChannelThreadForReply(L *glua.LState) int {
	r := lctx.getRobot(L, "PromptUserChannelThreadForReply")
	regexID := L.CheckString(2)
	user := L.CheckString(3)
	channel := L.CheckString(4)
	thread := L.CheckString(5)
	prompt := L.CheckString(6)

	if regexID == "" {
		L.RaiseError("PromptUserChannelThreadForReply: regexID must not be empty")
		return 0
	}
	if user == "" {
		L.RaiseError("PromptUserChannelThreadForReply: user must not be empty")
		return 0
	}
	if channel == "" {
		L.RaiseError("PromptUserChannelThreadForReply: channel must not be empty")
		return 0
	}
	// The Go code (as shown) ignores thread, but we pass it anyway for future or fixed usage.
	// No need to fail if thread == "".

	reply, ret := r.PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}
