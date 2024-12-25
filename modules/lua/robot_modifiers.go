package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// We’ll define a global map of valid string fields
var validStringFields = map[string]bool{
	"user":       true,
	"user_id":    true,
	"channel":    true,
	"channel_id": true,
	"thread_id":  true,
	"message_id": true,
	"plugin_id":  true,
	"protocol":   true,
	"brain":      true,
	"format":     true,
	// Add more if needed
}

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
	newFields["format"] = "fixed"
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

func (lctx *luaContext) botMessageFormat(L *glua.LState) int {
	ud := L.CheckUserData(1)
	formatVal := L.CheckString(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("MessageFormat")
		L.RaiseError("Invalid bot userdata for MessageFormat()")
		return 0
	}

	newFields := copyFields(lr.fields)
	newFields["format"] = formatVal
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

func copyFields(original map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range original {
		newMap[k] = v
	}
	return newMap
}

// newLuaBot creates a new Lua userdata for the bot with initialized fields and the "bot" metatable
func newLuaBot(L *glua.LState, r robot.Robot, fields map[string]interface{}) *glua.LUserData {
	newUD := L.NewUserData()
	newUD.Value = &luaRobot{r: r, fields: fields}
	// Assign the "bot" metatable
	L.SetMetatable(newUD, L.GetTypeMetatable("bot"))
	return newUD
}

// logBotErr logs an error specific to the bot userdata.
func (lctx *luaContext) logBotErr(caller string) {
	if lctx.r != nil {
		lctx.r.Log(robot.Error, fmt.Sprintf("%s called with invalid bot userdata", caller))
	} else {
		fmt.Printf("[ERR] %s called but robot is nil\n", caller)
	}
}
