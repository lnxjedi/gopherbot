package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterUtilMethods adds RandomInt, RandomString, Pause, CheckAdmin, Elevate, and Log to the bot's metatable.
func (lctx *luaContext) RegisterUtilMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"RandomInt":    lctx.botRandomInt,
		"RandomString": lctx.botRandomString,
		"Pause":        lctx.botPause,
		"CheckAdmin":   lctx.botCheckAdmin,
		"Elevate":      lctx.botElevate,
		"Log":          lctx.botLog,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// -------------------------------------------------------------------
// 1) bot:RandomInt(n)
// -------------------------------------------------------------------

// botRandomInt wraps r.RandomInt and returns a random integer up to n.
func (lctx *luaContext) botRandomInt(L *glua.LState) int {
	ud := L.CheckUserData(1)
	nLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("RandomInt")
		L.Push(glua.LNumber(0))
		return 1
	}

	if nLua.Type() != glua.LTNumber {
		lctx.Log(robot.Error, fmt.Sprintf("RandomInt requires a numeric argument, got %s", nLua.Type().String()))
		L.Push(glua.LNumber(0))
		return 1
	}

	n := int(nLua.(glua.LNumber))
	val := lr.r.RandomInt(n)
	L.Push(glua.LNumber(val))
	return 1
}

// -------------------------------------------------------------------
// 2) bot:RandomString(array)
// -------------------------------------------------------------------

// botRandomString implements r.RandomString(...) and returns a random string from the provided array.
func (lctx *luaContext) botRandomString(L *glua.LState) int {
	ud := L.CheckUserData(1)
	arrLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("RandomString")
		L.Push(glua.LString(""))
		return 1
	}

	if arrLua.Type() != glua.LTTable {
		lctx.Log(robot.Error, "RandomString called with non-table argument")
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
			lctx.Log(robot.Error, fmt.Sprintf("RandomString: non-string element at index %d, ignoring", i))
		}
	}

	if len(goSlice) == 0 {
		lctx.Log(robot.Error, "RandomString found no valid strings, returning empty string")
		L.Push(glua.LString(""))
		return 1
	}

	str := lr.r.RandomString(goSlice)
	L.Push(glua.LString(str))
	return 1
}

// -------------------------------------------------------------------
// 3) bot:Pause(seconds)
// -------------------------------------------------------------------

// botPause wraps r.Pause(...) and pauses execution for the specified number of seconds.
func (lctx *luaContext) botPause(L *glua.LState) int {
	ud := L.CheckUserData(1)
	secLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("Pause")
		return pushFail(L)
	}

	if secLua.Type() != glua.LTNumber {
		lctx.Log(robot.Error, fmt.Sprintf("Pause requires a numeric argument, got %s", secLua.Type().String()))
		return pushFail(L)
	}

	sec := float64(secLua.(glua.LNumber))
	lr.r.Pause(sec)
	return 0
}

// -------------------------------------------------------------------
// 4) bot:CheckAdmin() -> bool
// -------------------------------------------------------------------

// botCheckAdmin checks if the current user has administrative privileges.
func (lctx *luaContext) botCheckAdmin(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("CheckAdmin")
		L.Push(glua.LBool(false))
		return 1
	}

	isAdmin := lr.r.CheckAdmin()
	L.Push(glua.LBool(isAdmin))
	return 1
}

// -------------------------------------------------------------------
// 5) bot:Elevate(immediate) -> bool
// -------------------------------------------------------------------

// botElevate elevates the current user's privileges, optionally forcing a 2FA prompt.
func (lctx *luaContext) botElevate(L *glua.LState) int {
	ud := L.CheckUserData(1)
	immArg := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("Elevate")
		L.Push(glua.LBool(false))
		return 1
	}

	immediate := false
	if immArg.Type() == glua.LTBool {
		immediate = bool(immArg.(glua.LBool))
	} else {
		// If user didn't pass a bool, log and treat it as false
		lctx.Log(robot.Error, fmt.Sprintf("Elevate called with non-boolean argument (%s), defaulting to false", immArg.Type().String()))
	}

	success := lr.r.Elevate(immediate)
	L.Push(glua.LBool(success))
	return 1
}

// -------------------------------------------------------------------
// 6) bot:Log(level, message) -> no return
// -------------------------------------------------------------------

// botLog logs a message at the specified log level.
func (lctx *luaContext) botLog(L *glua.LState) int {
	ud := L.CheckUserData(1)
	levelArg := L.Get(2)
	msgArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("Log")
		return pushFail(L)
	}

	var lvl robot.LogLevel = robot.Info // default
	if levelArg.Type() == glua.LTNumber {
		numLevel := int(levelArg.(glua.LNumber))
		lvl = robot.LogLevel(numLevel)
	} else {
		lctx.Log(robot.Error, fmt.Sprintf("Log: expected numeric level, got %s; defaulting to Info", levelArg.Type().String()))
	}

	var msg string
	if msgArg.Type() == glua.LTString {
		msg = msgArg.String()
	} else {
		msg = fmt.Sprintf("Log called with non-string message type: %s", msgArg.Type().String())
		lctx.Log(robot.Error, msg)
		return pushFail(L)
	}

	lr.r.Log(lvl, msg)
	return 0
}
