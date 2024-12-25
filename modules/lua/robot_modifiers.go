package lua

import (
	"fmt"
	"strings"

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

// registerBotMetatableIfNeeded returns the "bot" metatable, creating it if needed.
func registerBotMetatableIfNeeded(L *glua.LState) *glua.LTable {
	mtVal := L.GetTypeMetatable("bot")
	if mtVal == glua.LNil {
		// Create the "bot" metatable
		mt := L.NewTypeMetatable("bot")

		// Use function-based __index so we can do:
		//   local val = bot.user  (from lr.fields)
		//   bot:Say("hi")        (method lookup)
		L.SetField(mt, "__index", L.NewFunction(botIndexFn))

		// Use function-based __newindex so we can do:
		//   bot.user = "david"
		L.SetField(mt, "__newindex", L.NewFunction(botNewIndexFn))

		return mt
	}

	// Otherwise, cast to *LTable
	mt, ok := mtVal.(*glua.LTable)
	if !ok {
		L.RaiseError("Expected 'bot' metatable to be *LTable, got %T", mtVal)
		return nil
	}
	return mt
}

// botIndexFn implements __index for bot userdatas.
func botIndexFn(L *glua.LState) int {
	// 1st arg is the userdata, 2nd is the key
	ud := L.CheckUserData(1)
	key := L.CheckString(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil {
		L.RaiseError("Invalid bot userdata in __index")
		return 0
	}

	// 1) Check if the key is one of our known fields (string or bool)
	if validStringFields[key] {
		// Retrieve the value
		if val, has := lr.fields[key]; has {
			strVal := fmt.Sprintf("%v", val)
			// Check if the key needs unwrapping
			if key == "user_id" || key == "channel_id" {
				strVal = strings.Trim(strVal, "<>")
			}
			L.Push(glua.LString(strVal))
			return 1
		}
		// Not set at all => empty string
		L.Push(glua.LString(""))
		return 1
	}

	if key == "threaded_message" {
		if val, ok := lr.fields["threaded_message"].(bool); ok && val {
			L.Push(glua.LBool(true))
		} else {
			L.Push(glua.LBool(false))
		}
		return 1
	}

	// 2) If not a field, then maybe it's a method in the metatable
	botVal := L.Get(1)              // LValue for the bot (userdata)
	mtVal := L.GetMetatable(botVal) // returns LValue (could be LNil or *LTable)
	if tbl, ok := mtVal.(*glua.LTable); ok {
		method := tbl.RawGetString(key) // or RawGet(glua.LString(key))
		if method != glua.LNil {
			// It's a function or something
			L.Push(method)
			return 1
		}
	}

	// 3) Not a recognized field or method -> return nil
	L.Push(glua.LNil)
	return 1
}

// botNewIndexFn implements __newindex for bot userdatas.
func botNewIndexFn(L *glua.LState) int {
	// 1st arg is the userdata, 2nd is the key, 3rd is the value
	ud := L.CheckUserData(1)
	key := L.CheckString(2)
	newVal := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil {
		L.RaiseError("Invalid bot userdata in __newindex")
		return 0
	}

	// 1) If key is in validStringFields
	if validStringFields[key] {
		// Must be a string
		if newVal.Type() != glua.LTString {
			L.RaiseError("Attempt to assign a non-string value to field '%s'", key)
			return 0
		}
		// Store it
		lr.fields[key] = newVal.String()
		return 0
	}

	// 2) If key == "threaded_message", must be bool
	if key == "threaded_message" {
		if b, isBool := newVal.(glua.LBool); isBool {
			lr.fields["threaded_message"] = bool(b)
		} else {
			L.RaiseError("Attempt to assign a non-boolean value to 'threaded_message'")
		}
		return 0
	}

	// 3) Otherwise, raise error: not an allowed field
	L.RaiseError("Field '%s' is not a recognized bot field", key)
	return 0
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
	newFields["format"] = "FixedFormat"
	newUD := newLuaBot(L, lr.r, newFields)
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
	// e.g. newFields["direct"] = true
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
