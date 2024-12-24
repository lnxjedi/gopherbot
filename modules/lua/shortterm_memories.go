package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterShortTermMemoryMethods adds ephemeral memory methods to the robot's metatable:
//
//	Remember(key, value, shared)
//	RememberThread(key, value, shared)
//	RememberContext(context, value)
//	RememberContextThread(context, value)
//	Recall(key, shared) -> string
func (lctx luaContext) RegisterShortTermMemoryMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"Remember":              lctx.robotRemember,
		"RememberThread":        lctx.robotRememberThread,
		"RememberContext":       lctx.robotRememberContext,
		"RememberContextThread": lctx.robotRememberContextThread,
		"Recall":                lctx.robotRecall,
	}
	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}

// -------------------------------------------------------------------
// 1) robot:Remember(key, value, shared)
// -------------------------------------------------------------------

// robotRemember allows Lua scripts to remember a key-value pair with an optional shared flag.
func (lctx luaContext) robotRemember(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	val := L.Get(3)
	sharedArg := L.Get(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("Remember")
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
		// Default to false if argument is missing or not a bool
		shared = false
	}

	lr.r.Remember(key.String(), val.String(), shared)
	return pushFail(L)
}

// -------------------------------------------------------------------
// 2) robot:RememberThread(key, value, shared)
// -------------------------------------------------------------------

// robotRememberThread allows Lua scripts to remember a key-value pair in a threaded context with an optional shared flag.
func (lctx luaContext) robotRememberThread(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	val := L.Get(3)
	sharedArg := L.Get(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("RememberThread")
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
// 3) robot:RememberContext(context, value)
// -------------------------------------------------------------------

// robotRememberContext allows Lua scripts to remember a value within a specific context.
func (lctx luaContext) robotRememberContext(L *glua.LState) int {
	ud := L.CheckUserData(1)
	cArg := L.Get(2)
	vArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("RememberContext")
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
// 4) robot:RememberContextThread(context, value)
// -------------------------------------------------------------------

// robotRememberContextThread allows Lua scripts to remember a value within a specific context in a threaded environment.
func (lctx luaContext) robotRememberContextThread(L *glua.LState) int {
	ud := L.CheckUserData(1)
	cArg := L.Get(2)
	vArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("RememberContextThread")
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
// 5) robot:Recall(key, shared) -> string
// -------------------------------------------------------------------

// robotRecall allows Lua scripts to recall a value by key with an optional shared flag.
func (lctx luaContext) robotRecall(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	sharedArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logErr("Recall")
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
