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

func copyFields(original map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range original {
		newMap[k] = v
	}
	return newMap
}

// -------------------------------------------------------------------
// The rest: your existing bot modifiers (Clone, Fixed, etc.)
// -------------------------------------------------------------------

func (lctx *luaContext) botClone(L *glua.LState) int {
	lr, ok := lctx.getRobotUD(L, "Clone")
	if !ok {
		L.RaiseError("invalid robot userdata in Clone")
		return 0
	}

	newFields := copyFields(lr.fields)
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

func (lctx *luaContext) botFixed(L *glua.LState) int {
	lr, ok := lctx.getRobotUD(L, "Fixed")
	if !ok {
		L.RaiseError("invalid robot userdata in Fixed")
		return 0
	}

	newFields := copyFields(lr.fields)
	fixedBot := lr.r.Fixed()
	newUD := newLuaBot(L, fixedBot, newFields)
	L.Push(newUD)
	return 1
}

func (lctx *luaContext) botDirect(L *glua.LState) int {
	lr, ok := lctx.getRobotUD(L, "Direct")
	if !ok {
		L.RaiseError("invalid robot userdata in Direct")
		return 0
	}

	newFields := copyFields(lr.fields)
	newFields["channel"] = ""
	newFields["channel_id"] = ""
	newFields["thread_id"] = ""
	newFields["threaded"] = false
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

func (lctx *luaContext) botThreaded(L *glua.LState) int {
	lr, ok := lctx.getRobotUD(L, "Threaded")
	if !ok {
		L.RaiseError("invalid robot userdata in Threaded")
		return 0
	}
	// Technically an error, but *shrug*
	if lr.fields["channel"] == "" {
		L.Push(newLuaBot(L, lr.r, lr.fields))
		return 1
	}

	newFields := copyFields(lr.fields)
	newFields["threaded_message"] = true
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// botMessageFormat updates the message format of the bot.
func (lctx *luaContext) botMessageFormat(L *glua.LState) int {
	lr, ok := lctx.getRobotUD(L, "MessageFormat")
	if !ok {
		L.RaiseError("invalid robot userdata in MessageFormat")
		return 0
	}
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
	updatedRobot := lr.r.MessageFormat(robot.MessageFormat(formatInt))

	newFields := copyFields(lr.fields)

	// Create a new Lua bot userdata with the updated robot and fields
	newUD := newLuaBot(L, updatedRobot, newFields)
	L.Push(newUD)
	return 1
}
