package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterUtilMethods adds RandomInt, RandomString, Pause, CheckAdmin, Elevate, and Log to the robot's metatable
func RegisterUtilMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"RandomInt":    robotRandomInt,
		"RandomString": robotRandomString,
		"Pause":        robotPause,
		"CheckAdmin":   robotCheckAdmin,
		"Elevate":      robotElevate,
		"Log":          robotLog,
	}
	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}

// robotRandomInt wraps r.RandomInt
func robotRandomInt(L *glua.LState) int {
	ud := L.CheckUserData(1)
	nLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		if lr != nil && lr.r != nil {
			lr.r.Log(robot.Error, "RandomInt called with invalid robot userdata")
		}
		L.Push(glua.LNumber(0))
		return 1
	}

	if nLua.Type() != glua.LTNumber {
		lr.r.Log(robot.Error, fmt.Sprintf("RandomInt requires a numeric argument, got %s", nLua.Type().String()))
		L.Push(glua.LNumber(0))
		return 1
	}

	n := int(nLua.(glua.LNumber))
	val := lr.r.RandomInt(n)
	L.Push(glua.LNumber(val))
	return 1
}

// robotRandomString implements r.RandomString(...)
func robotRandomString(L *glua.LState) int {
	ud := L.CheckUserData(1)
	arrLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		if lr != nil && lr.r != nil {
			lr.r.Log(robot.Error, "RandomString called with invalid robot userdata")
		}
		L.Push(glua.LString(""))
		return 1
	}

	if arrLua.Type() != glua.LTTable {
		lr.r.Log(robot.Error, "RandomString called with non-table argument")
		L.Push(glua.LString(""))
		return 1
	}

	tbl := arrLua.(*glua.LTable)
	length := tbl.Len()

	var goSlice []string
	for i := 1; i <= length; i++ {
		val := tbl.RawGetInt(i)
		if val.Type() == glua.LTString {
			goSlice = append(goSlice, val.String())
		} else {
			lr.r.Log(robot.Error, fmt.Sprintf("RandomString: non-string element at index %d, ignoring", i))
		}
	}

	if len(goSlice) == 0 {
		lr.r.Log(robot.Error, "RandomString found no valid strings, returning empty string")
		L.Push(glua.LString(""))
		return 1
	}

	str := lr.r.RandomString(goSlice)
	L.Push(glua.LString(str))
	return 1
}

// robotPause wraps r.Pause(...)
func robotPause(L *glua.LState) int {
	ud := L.CheckUserData(1)
	secLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		if lr != nil && lr.r != nil {
			lr.r.Log(robot.Error, "Pause called with invalid robot userdata")
		}
		return 0
	}

	if secLua.Type() != glua.LTNumber {
		lr.r.Log(robot.Error, fmt.Sprintf("Pause requires a numeric argument, got %s", secLua.Type().String()))
		return 0
	}

	sec := float64(secLua.(glua.LNumber))
	lr.r.Pause(sec)
	return 0
}

// robot:CheckAdmin() -> bool
func robotCheckAdmin(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		if lr != nil && lr.r != nil {
			lr.r.Log(robot.Error, "CheckAdmin called with invalid robot userdata")
		}
		L.Push(glua.LBool(false))
		return 1
	}

	isAdmin := lr.r.CheckAdmin()
	L.Push(glua.LBool(isAdmin))
	return 1
}

// robot:Elevate(immediate) -> bool
// immediate is a Lua boolean indicating whether to forcibly prompt for 2fa
func robotElevate(L *glua.LState) int {
	ud := L.CheckUserData(1)
	immArg := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		if lr != nil && lr.r != nil {
			lr.r.Log(robot.Error, "Elevate called with invalid robot userdata")
		}
		L.Push(glua.LBool(false))
		return 1
	}

	immediate := false
	if immArg.Type() == glua.LTBool {
		immediate = bool(immArg.(glua.LBool))
	} else {
		// If user didn't pass a bool, log and treat it as false
		lr.r.Log(robot.Error, fmt.Sprintf("Elevate called with non-boolean argument (%s), defaulting to false", immArg.Type().String()))
	}

	success := lr.r.Elevate(immediate)
	L.Push(glua.LBool(success))
	return 1
}

// robot:Log(level, message) -> no return
// level: numeric log level (logTrace=0, logDebug=1, logInfo=2, etc.)
// message: string
func robotLog(L *glua.LState) int {
	ud := L.CheckUserData(1)
	levelArg := L.Get(2)
	msgArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		if lr != nil && lr.r != nil {
			lr.r.Log(robot.Error, "Log called with invalid robot userdata")
		}
		return 0
	}

	var lvl robot.LogLevel = robot.Info // default
	if levelArg.Type() == glua.LTNumber {
		numLevel := int(levelArg.(glua.LNumber))
		lvl = robot.LogLevel(numLevel)
	} else {
		lr.r.Log(robot.Error, fmt.Sprintf("Log: expected numeric level, got %s; defaulting to Info", levelArg.Type().String()))
	}

	var msg string
	if msgArg.Type() == glua.LTString {
		msg = msgArg.String()
	} else {
		msg = fmt.Sprintf("Log called with non-string message type: %s", msgArg.Type().String())
		lr.r.Log(robot.Error, msg)
		return 0
	}

	lr.r.Log(lvl, msg)
	return 0
}
