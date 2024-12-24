package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// newLuaRobot creates a new Lua userdata for the robot.
// This function remains unchanged as it serves as a helper.
func newLuaRobot(L *glua.LState, r robot.Robot, env map[string]string) *glua.LUserData {
	newUD := L.NewUserData()
	newUD.Value = &luaRobot{r: r, env: env}
	// Set the same metatable so we can still call :Say, :Reply, etc.
	L.SetMetatable(newUD, L.GetTypeMetatable("robot"))
	return newUD
}

// robotNew allows for a more natural bot = robot:New()
func (lctx luaContext) robotNew(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("New")
		return pushFail(L)
	}

	newUD := newLuaRobot(L, lr.r, lr.env)
	L.Push(newUD)
	return 1
}

// robotDirect creates a direct instance of the robot.
func (lctx luaContext) robotDirect(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("Direct")
		return pushFail(L)
	}

	newR := lr.r.Direct()
	newUD := newLuaRobot(L, newR, lr.env)

	// Return the new userdata to Lua
	L.Push(newUD)
	return 1
}

// robotThreaded creates a threaded instance of the robot.
func (lctx luaContext) robotThreaded(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("Threaded")
		return pushFail(L)
	}

	newR := lr.r.Threaded()
	newUD := newLuaRobot(L, newR, lr.env)
	L.Push(newUD)
	return 1
}

// robotFixed creates a fixed instance of the robot.
func (lctx luaContext) robotFixed(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("Fixed")
		return pushFail(L)
	}

	newR := lr.r.Fixed()
	newUD := newLuaRobot(L, newR, lr.env)
	L.Push(newUD)
	return 1
}

// robotMessageFormat sets the message format for the robot.
func (lctx luaContext) robotMessageFormat(L *glua.LState) int {
	ud := L.CheckUserData(1)
	formatVal := L.CheckNumber(2) // e.g., 0=Raw,1=Fixed,2=Variable, etc.

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("MessageFormat")
		return pushFail(L)
	}

	format := robot.MessageFormat(int(formatVal))
	newR := lr.r.MessageFormat(format)

	newUD := newLuaRobot(L, newR, lr.env)
	L.Push(newUD)
	return 1
}

// RegisterRobotModifiers registers all robot modifier methods with Lua.
func (lctx luaContext) RegisterRobotModifiers(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"New":           lctx.robotNew,
		"Fixed":         lctx.robotFixed,
		"Direct":        lctx.robotDirect,
		"Threaded":      lctx.robotThreaded,
		"MessageFormat": lctx.robotMessageFormat,
	}
	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}
