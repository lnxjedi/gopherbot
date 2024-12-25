package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterShortTermMemoryMethods adds ephemeral memory methods to the bot's metatable:
//
//	bot:Remember(key, value, shared)
//	bot:RememberThread(key, value, shared)
//	bot:RememberContext(context, value)
//	bot:RememberContextThread(context, value)
//	bot:Recall(key, shared) -> string
func (lctx luaContext) RegisterShortTermMemoryMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"Remember":              lctx.botRemember,
		"RememberThread":        lctx.botRememberThread,
		"RememberContext":       lctx.botRememberContext,
		"RememberContextThread": lctx.botRememberContextThread,
		"Recall":                lctx.botRecall,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// -------------------------------------------------------------------
// 1) bot:Remember(key, value, shared)
// -------------------------------------------------------------------

// botRemember allows Lua scripts to remember a key-value pair with an optional shared flag.
func (lctx luaContext) botRemember(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	val := L.Get(3)
	sharedArg := L.Get(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("Remember")
		return pushFail(L)
	}

	// Validate arguments
	if key.Type() != glua.LTString || val.Type() != glua.LTString {
		lr.r.Log(robot.Error, "Remember: key and value must be strings")
		return pushFail(L)
	}
	var shared bool
	if sharedArg.Type() == glua.LTBool {
		shared = bool(sharedArg.(glua.LBool))
	} else {
		// Default to false
		shared = false
	}

	lr.r.Remember(key.String(), val.String(), shared)
	return pushFail(L)
}

// -------------------------------------------------------------------
// 2) bot:RememberThread(key, value, shared)
// -------------------------------------------------------------------

// botRememberThread remembers a key-value pair in a threaded context with an optional shared flag.
func (lctx luaContext) botRememberThread(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	val := L.Get(3)
	sharedArg := L.Get(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("RememberThread")
		return pushFail(L)
	}

	// Validate arguments
	if key.Type() != glua.LTString || val.Type() != glua.LTString {
		lr.r.Log(robot.Error, "RememberThread: key and value must be strings")
		return pushFail(L)
	}
	var shared bool
	if sharedArg.Type() == glua.LTBool {
		shared = bool(sharedArg.(glua.LBool))
	} else {
		shared = false
	}

	lr.r.RememberThread(key.String(), val.String(), shared)
	return pushFail(L)
}

// -------------------------------------------------------------------
// 3) bot:RememberContext(context, value)
// -------------------------------------------------------------------

// botRememberContext remembers a value within a specific context.
func (lctx luaContext) botRememberContext(L *glua.LState) int {
	ud := L.CheckUserData(1)
	cArg := L.Get(2)
	vArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("RememberContext")
		return pushFail(L)
	}

	// Validate arguments
	if cArg.Type() != glua.LTString || vArg.Type() != glua.LTString {
		lr.r.Log(robot.Error, "RememberContext: context and value must be strings")
		return pushFail(L)
	}

	lr.r.RememberContext(cArg.String(), vArg.String())
	return pushFail(L)
}

// -------------------------------------------------------------------
// 4) bot:RememberContextThread(context, value)
// -------------------------------------------------------------------

// botRememberContextThread remembers a value within a specific context in a threaded environment.
func (lctx luaContext) botRememberContextThread(L *glua.LState) int {
	ud := L.CheckUserData(1)
	cArg := L.Get(2)
	vArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("RememberContextThread")
		return pushFail(L)
	}

	// Validate arguments
	if cArg.Type() != glua.LTString || vArg.Type() != glua.LTString {
		lr.r.Log(robot.Error, "RememberContextThread: context and value must be strings")
		return pushFail(L)
	}

	lr.r.RememberContextThread(cArg.String(), vArg.String())
	return pushFail(L)
}

// -------------------------------------------------------------------
// 5) bot:Recall(key, shared) -> string
// -------------------------------------------------------------------

// botRecall recalls a value by key with an optional shared flag.
func (lctx luaContext) botRecall(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	sharedArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("Recall")
		L.Push(glua.LString(""))
		return 1
	}

	// Validate key argument
	if key.Type() != glua.LTString {
		lr.r.Log(robot.Error, "Recall: key must be a string")
		L.Push(glua.LString(""))
		return 1
	}

	var shared bool
	if sharedArg.Type() == glua.LTBool {
		shared = bool(sharedArg.(glua.LBool))
	} else {
		shared = false
	}

	value := lr.r.Recall(key.String(), shared)
	L.Push(glua.LString(value))
	return 1
}
