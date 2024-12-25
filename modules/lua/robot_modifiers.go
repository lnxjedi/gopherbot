// robot_modifiers.go
package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterRobotModifiers registers all bot modifier methods with the "bot" metatable.
func (lctx *luaContext) RegisterRobotModifiers(L *glua.LState) {
	// Merge these methods into the metatable
	methods := map[string]glua.LGFunction{
		"Clone":         lctx.botClone,
		"Fixed":         lctx.botFixed,
		"Direct":        lctx.botDirect,
		"Threaded":      lctx.botThreaded,
		"MessageFormat": lctx.botMessageFormat,
		// Add other bot methods here as needed
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// registerBotMetatableIfNeeded returns the "bot" metatable, creating if needed.
func registerBotMetatableIfNeeded(L *glua.LState) *glua.LTable {
	// L.GetTypeMetatable returns an LValue, which may be LNil or an *LTable.
	mtVal := L.GetTypeMetatable("bot")

	// If we haven't created the "bot" metatable yet ...
	if mtVal == glua.LNil {
		// Create a fresh metatable
		newMT := L.NewTypeMetatable("bot")

		// Set __index = newMT so that "bot:SomeMethod" will look up
		// SomeMethod in the newMT table
		L.SetField(newMT, "__index", newMT)

		return newMT // newMT is already *LTable
	}

	// Otherwise, mtVal should be *LTable
	mt, ok := mtVal.(*glua.LTable)
	if !ok {
		L.RaiseError("Expected *LTable for 'bot' metatable, got %T", mtVal)
		return nil
	}
	return mt
}

// botClone creates a new bot userdata instance by cloning the current bot's fields
func (lctx *luaContext) botClone(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("Clone")
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
		lctx.logBotErr("Fixed")
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
		lctx.logBotErr("Direct")
		L.RaiseError("Invalid bot userdata for Direct()")
		return 0
	}

	// Copy existing fields
	newFields := copyFields(lr.fields)

	// Potentially modify fields specific to Direct() (example: set a "direct" flag)
	// newFields["direct"] = true

	// Create a new bot userdata
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// botThreaded creates a threaded instance of the bot by copying existing fields and setting threaded_message = true
func (lctx *luaContext) botThreaded(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("Threaded")
		L.RaiseError("Invalid bot userdata for Threaded()")
		return 0
	}

	// Copy existing fields
	newFields := copyFields(lr.fields)
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
		lctx.logBotErr("MessageFormat")
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

// copyFields creates a shallow copy of the fields map
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
