package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterRobotModifiers registers all bot modifier methods with the "bot" metatable.
func (lctx *luaContext) RegisterRobotModifiers(L *glua.LState) {
	mt := registerBotMetatableIfNeeded(L)

	methods := map[string]glua.LGFunction{
		"Fixed":         lctx.botFixed,
		"Direct":        lctx.botDirect,
		"Threaded":      lctx.botThreaded,
		"MessageFormat": lctx.botMessageFormat,
	}
	L.SetFuncs(mt, methods)
}

func (lctx *luaContext) botFixed(L *glua.LState) int {
	lr := lctx.getRobotUD(L, "Fixed")

	fixedBot := lr.r.Fixed()
	newUD := lctx.newLuaBot(L, fixedBot)
	L.Push(newUD)
	return 1
}

func (lctx *luaContext) botDirect(L *glua.LState) int {
	lr := lctx.getRobotUD(L, "Direct")

	directBot := lr.r.Direct()
	newUD := lctx.newLuaBot(L, directBot)
	robot, _ := newUD.Value.(*luaRobot)
	robot.fields["channel"] = ""
	robot.fields["channel_id"] = ""
	L.Push(newUD)
	return 1
}

func (lctx *luaContext) botThreaded(L *glua.LState) int {
	lr := lctx.getRobotUD(L, "Threaded")

	threadedBot := lr.r.Threaded()
	newUD := lctx.newLuaBot(L, threadedBot)
	robot, _ := newUD.Value.(*luaRobot)
	robot.fields["threaded"] = true
	L.Push(newUD)
	return 1
}

// botMessageFormat updates the message format of the bot.
func (lctx *luaContext) botMessageFormat(L *glua.LState) int {
	lr := lctx.getRobotUD(L, "MessageFormat")
	formatArg := L.Get(2)

	// Validate that formatArg is a number
	if formatArg.Type() != glua.LTNumber {
		L.RaiseError("MessageFormat requires a numeric argument (use fmt.* constants)")
		return 0
	}

	// Convert to integer
	formatInt := int(formatArg.(glua.LNumber))

	// Validate the format value
	if !isValidMessageFormat(formatInt) {
		L.RaiseError("Invalid MessageFormat value: %d. Must be Raw=0, Fixed=1, or Variable=2", formatInt)
		return 0
	}

	// Update the robot's message format
	formattedRobot := lr.r.MessageFormat(robot.MessageFormat(formatInt))
	newUD := lctx.newLuaBot(L, formattedRobot)
	L.Push(newUD)
	return 1
}
