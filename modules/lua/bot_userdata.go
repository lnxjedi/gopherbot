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
	"protocol":   true,
	"brain":      true,
}

// initializeFields initializes the robot fields from bot.
func initializeFields(env map[string]string) (map[string]interface{}, error) {
	fields := make(map[string]interface{})

	for key := range validStringFields {
		if val, exists := env[key]; exists {
			fields[key] = val
		} else {
			fields[key] = "" // Default to empty string if not set
		}
	}

	// Handle threaded_message as boolean
	threadedVal, exists := env["threaded_message"]
	if exists && strings.ToLower(threadedVal) == "true" {
		fields["threaded_message"] = true
	} else {
		fields["threaded_message"] = false
	}

	return fields, nil
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
