// robot_modifiers.go
package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterRobotModifiers registers all bot modifier methods with the bot metatable.
func (lctx *luaContext) RegisterRobotModifiers(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"Clone":         lctx.botClone,
		"Fixed":         lctx.botFixed,
		"Direct":        lctx.botDirect,
		"Threaded":      lctx.botThreaded,
		"MessageFormat": lctx.botMessageFormat,
		// Add other bot methods here as needed
	}
	botIndex := getBotMethodTable(L)
	L.SetFuncs(botIndex, methods)
}

// getBotMethodTable retrieves the methods table from the bot metatable
func getBotMethodTable(L *glua.LState) *glua.LTable {
	// Retrieve the metatable associated with type "bot"
	mt := L.GetTypeMetatable("bot")
	if mt == glua.LNil {
		// If "bot" metatable doesn't exist, create it
		mt = L.NewTypeMetatable("bot")
		L.SetMetatable(mt, mt)
	}

	// Get the "methods" table from the metatable
	methods := L.GetField(mt, "methods")
	if tbl, ok := methods.(*glua.LTable); ok {
		return tbl
	}

	// If "methods" table doesn't exist, create it
	tbl := L.NewTable()
	L.SetField(mt, "methods", tbl)
	return tbl
}

// botClone creates a new bot userdata instance by cloning the current bot's fields
func (lctx *luaContext) botClone(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("Clone")
		L.RaiseError("Invalid bot userdata for Clone()")
		return 0
	}

	// Copy existing fields
	newFields := copyFields(lr.fields)

	// Create a new bot userdata
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// botFixed creates a fixed instance of the bot by copying existing fields and modifying 'format'
func (lctx *luaContext) botFixed(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("Fixed")
		L.RaiseError("Invalid bot userdata for Fixed()")
		return 0
	}

	// Copy existing fields
	newFields := copyFields(lr.fields)

	// Modify the 'format' field as needed (example: set to "FixedFormat")
	newFields["format"] = "FixedFormat"

	// Create a new bot userdata
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// botDirect creates a direct instance of the bot by copying existing fields
func (lctx *luaContext) botDirect(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("Direct")
		L.RaiseError("Invalid bot userdata for Direct()")
		return 0
	}

	// Copy existing fields
	newFields := copyFields(lr.fields)

	// Potentially modify fields specific to Direct() (example: set a specific flag)
	// newFields["direct"] = true

	// Create a new bot userdata
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// botThreaded creates a threaded instance of the bot by copying existing fields and setting threaded_message to true
func (lctx *luaContext) botThreaded(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("Threaded")
		L.RaiseError("Invalid bot userdata for Threaded()")
		return 0
	}

	// Copy existing fields
	newFields := copyFields(lr.fields)

	// Set 'threaded_message' to true
	newFields["threaded_message"] = true

	// Create a new bot userdata
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// botMessageFormat sets the message format for the bot by copying existing fields and modifying 'format'
func (lctx *luaContext) botMessageFormat(L *glua.LState) int {
	ud := L.CheckUserData(1)
	formatVal := L.CheckString(2) // Expecting a string for format

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logErr("MessageFormat")
		L.RaiseError("Invalid bot userdata for MessageFormat()")
		return 0
	}

	// Copy existing fields
	newFields := copyFields(lr.fields)

	// Set the new format
	newFields["format"] = formatVal

	// Create a new bot userdata
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// copyFields creates a deep copy of the fields map
func copyFields(original map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range original {
		newMap[k] = v
	}
	return newMap
}

// newLuaBot creates a new Lua userdata for the bot with initialized fields and appropriate metatable.
func newLuaBot(L *glua.LState, r robot.Robot, fields map[string]interface{}) *glua.LUserData {
	newUD := L.NewUserData()
	newUD.Value = &luaRobot{r: r, fields: fields}
	// Set the metatable so we can call methods like :Clone, :Say, etc.
	L.SetMetatable(newUD, L.GetTypeMetatable("bot"))
	return newUD
}

// logErr logs an error with the saved bot if non-nil, otherwise prints to stdout
func (lctx *luaContext) logErr(caller string) {
	if lctx.r != nil {
		lctx.r.Log(robot.Error, fmt.Sprintf("%s called with invalid robot userdata", caller))
	} else {
		fmt.Printf("[ERR] %s called but robot is nil\n", caller)
	}
}
