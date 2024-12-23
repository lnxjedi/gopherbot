package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

func newLuaRobot(L *glua.LState, r robot.Robot) *glua.LUserData {
	newUD := L.NewUserData()
	newUD.Value = &luaRobot{r: r}
	// Set the same metatable so we can still call :Say, :Reply, etc.
	L.SetMetatable(newUD, L.GetTypeMetatable("robot"))
	return newUD
}

// robotNew allows for a more natural bot = robot:New()
func robotNew(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logErr(lr, "New")
		return 0
	}

	newUD := newLuaRobot(L, lr.r)

	// 3. Return that new userdata to Lua
	L.Push(newUD)
	return 1
}

func robotDirect(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logErr(lr, "Direct")
		return 0
	}

	// 1. Call Go's r.Direct() to get a new Robot
	newR := lr.r.Direct()

	// 2. Wrap it in new userdata
	newUD := newLuaRobot(L, newR)

	// 3. Return that new userdata to Lua
	L.Push(newUD)
	return 1
}

func robotThreaded(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logErr(lr, "Threaded")
		return 0
	}

	newR := lr.r.Threaded()
	newUD := newLuaRobot(L, newR)
	L.Push(newUD)
	return 1
}

func robotFixed(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logErr(lr, "Fixed")
		return 0
	}

	newR := lr.r.Fixed()
	newUD := newLuaRobot(L, newR)
	L.Push(newUD)
	return 1
}

func robotMessageFormat(L *glua.LState) int {
	ud := L.CheckUserData(1)
	formatVal := L.CheckNumber(2) // e.g., 0=Raw,1=Fixed,2=Variable, etc.

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		logErr(lr, "MessageFormat")
		return 0
	}

	format := robot.MessageFormat(int(formatVal))
	newR := lr.r.MessageFormat(format)

	newUD := newLuaRobot(L, newR)
	L.Push(newUD)
	return 1
}

func RegisterRobotModifiers(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"New":           robotNew,
		"Fixed":         robotFixed,
		"Direct":        robotDirect,
		"Threaded":      robotThreaded,
		"MessageFormat": robotMessageFormat,
	}
	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}
