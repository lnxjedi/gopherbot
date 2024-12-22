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
func RegisterShortTermMemoryMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"Remember":              robotRemember,
		"RememberThread":        robotRememberThread,
		"RememberContext":       robotRememberContext,
		"RememberContextThread": robotRememberContextThread,
		"Recall":                robotRecall,
	}
	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}

// -------------------------------------------------------------------
// 1) robot:Remember(key, value, shared)
// -------------------------------------------------------------------
func robotRemember(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	val := L.Get(3)
	sharedArg := L.Get(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "Remember")
		return 0
	}

	// Validate arguments
	if key.Type() != glua.LTString || val.Type() != glua.LTString {
		lr.r.Log(robot.Error, "Remember: key and value must be strings")
		return 0
	}
	var shared bool
	if sharedArg.Type() == glua.LTBool {
		shared = bool(sharedArg.(glua.LBool))
	} else {
		// Default to false if argument is missing or not a bool
		shared = false
	}

	lr.r.Remember(key.String(), val.String(), shared)
	return 0
}

// -------------------------------------------------------------------
// 2) robot:RememberThread(key, value, shared)
// -------------------------------------------------------------------
func robotRememberThread(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	val := L.Get(3)
	sharedArg := L.Get(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "RememberThread")
		return 0
	}

	if key.Type() != glua.LTString || val.Type() != glua.LTString {
		lr.r.Log(robot.Error, "RememberThread: key and value must be strings")
		return 0
	}
	var shared bool
	if sharedArg.Type() == glua.LTBool {
		shared = bool(sharedArg.(glua.LBool))
	} else {
		shared = false
	}

	lr.r.RememberThread(key.String(), val.String(), shared)
	return 0
}

// -------------------------------------------------------------------
// 3) robot:RememberContext(context, value)
// -------------------------------------------------------------------
func robotRememberContext(L *glua.LState) int {
	ud := L.CheckUserData(1)
	cArg := L.Get(2)
	vArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "RememberContext")
		return 0
	}

	if cArg.Type() != glua.LTString || vArg.Type() != glua.LTString {
		lr.r.Log(robot.Error, "RememberContext: context and value must be strings")
		return 0
	}

	lr.r.RememberContext(cArg.String(), vArg.String())
	return 0
}

// -------------------------------------------------------------------
// 4) robot:RememberContextThread(context, value)
// -------------------------------------------------------------------
func robotRememberContextThread(L *glua.LState) int {
	ud := L.CheckUserData(1)
	cArg := L.Get(2)
	vArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "RememberContextThread")
		return 0
	}

	if cArg.Type() != glua.LTString || vArg.Type() != glua.LTString {
		lr.r.Log(robot.Error, "RememberContextThread: context and value must be strings")
		return 0
	}

	lr.r.RememberContextThread(cArg.String(), vArg.String())
	return 0
}

// -------------------------------------------------------------------
// 5) robot:Recall(key, shared) -> string
// -------------------------------------------------------------------
func robotRecall(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.Get(2)
	sharedArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "Recall")
		L.Push(glua.LString(""))
		return 1
	}

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
