package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterRobotModifiers registers all bot modifier methods with the "bot" metatable.
func (lctx *luaContext) RegisterRobotModifiers(L *glua.LState) {
	mt := registerBotMetatableIfNeeded(L)

	methods := map[string]glua.LGFunction{
		"Clone":         lctx.botClone,
		"Fixed":         lctx.botFixed,
		"Direct":        lctx.botDirect,
		"Threaded":      lctx.botThreaded,
		"MessageFormat": lctx.botMessageFormat,
	}
	L.SetFuncs(mt, methods)
}

// -------------------------------------------------------------------
// The rest: your existing bot modifiers (Clone, Fixed, etc.)
// -------------------------------------------------------------------

func (lctx *luaContext) botClone(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("Clone")
		L.RaiseError("Invalid bot userdata for Clone()")
		return 0
	}

	newFields := copyFields(lr.fields)
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

func (lctx *luaContext) botFixed(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("Fixed")
		L.RaiseError("Invalid bot userdata for Fixed()")
		return 0
	}

	newFields := copyFields(lr.fields)
	newFields["format"] = "Fixed"
	fixedBot := lr.r.Fixed()
	newUD := newLuaBot(L, fixedBot, newFields)
	L.Push(newUD)
	return 1
}

func (lctx *luaContext) botDirect(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("Direct")
		L.RaiseError("Invalid bot userdata for Direct()")
		return 0
	}

	newFields := copyFields(lr.fields)
	newFields["channel"] = ""
	newFields["thread_id"] = ""
	newFields["threaded"] = false
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

func (lctx *luaContext) botThreaded(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("Threaded")
		L.RaiseError("Invalid bot userdata for Threaded()")
		return 0
	}

	newFields := copyFields(lr.fields)
	newFields["threaded_message"] = true
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// botMessageFormat updates the message format of the bot.
func (lctx *luaContext) botMessageFormat(L *glua.LState) int {
	ud := L.CheckUserData(1)
	formatArg := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("MessageFormat")
		return 0
	}

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
	updatedRobot := lr.r.MessageFormat(robot.MessageFormat(formatInt))

	// Update the fields with the new format string
	formatStr := ""
	switch robot.MessageFormat(formatInt) {
	case robot.Raw:
		formatStr = "Raw"
	case robot.Fixed:
		formatStr = "Fixed"
	case robot.Variable:
		formatStr = "Variable"
	}

	newFields := copyFields(lr.fields)
	newFields["format"] = formatStr

	// Create a new Lua bot userdata with the updated robot and fields
	newUD := newLuaBot(L, updatedRobot, newFields)
	L.Push(newUD)
	return 1
}
