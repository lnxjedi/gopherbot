package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// luaRobot holds a reference to the robot.Robot interface, so we can
// call methods on the underlying Gopherbot Robot from Lua.
type luaRobot struct {
	r   robot.Robot
	env map[string]string
}

// CallExtension loads and executes a Lua script at taskPath, passing in:
//   - env: a map of environment-like variables to expose in Lua as a read-only table
//   - r: the robot.Robot
//   - privileged: if true, we don't remove any functions; if false, we remove os.getenv, os.setenv
//   - args: the script arguments
func CallExtension(taskPath, taskName string, env map[string]string, r robot.Robot, privileged bool, args []string) (robot.TaskRetVal, error) {
	L := glua.NewState()
	defer L.Close()

	// This is done automatically unless the SkipOpenLibs option is passed
	// L.OpenLibs()

	// Replace os.getenv with our version that uses the env map, replace
	// os.setenv with a noop version that logs a warning.
	modifyOSFunctions(L, r, env)

	// Register the "robot" type and any base methods.
	registerRobotType(L)

	// Register additional sets (e.g., message methods):
	RegisterMessageMethods(L)
	RegisterRobotModifiers(L)
	RegisterLongtermMemoryMethods(L)
	RegisterConfigMethod(L)
	RegisterUtilMethods(L)
	RegisterAttributeMethods(L)
	// RegisterLogMethods(L)

	// Create the robot userdata object and set it as "robot".
	robotUD := L.NewUserData()
	robotUD.Value = &luaRobot{r: r, env: env}
	L.SetMetatable(robotUD, L.GetTypeMetatable("robot"))
	L.SetGlobal("robot", robotUD)

	// Register constants (RetVal/ret, TaskRetVal/task, etc.).
	registerConstants(L)

	// Provide the script arguments as a Lua table "arg", similar to
	// a standard Lua script.
	argsTable := L.CreateTable(len(args), 0)
	argsTable.RawSetInt(0, glua.LString(taskName))
	for i, a := range args {
		argsTable.RawSetInt(i+1, glua.LString(a))
	}
	L.SetGlobal("arg", argsTable)

	// Compile and run the Lua file.
	if err := L.DoFile(taskPath); err != nil {
		r.Log(robot.Error, fmt.Sprintf("Lua error in script '%s': %v", taskName, err))
		return robot.MechanismFail, err
	}

	// 9. Check the script’s return value (default to Normal).
	retVal := L.Get(-1) // top of stack
	L.Pop(1)

	var taskReturn robot.TaskRetVal = robot.Normal
	if ln, ok := retVal.(glua.LNumber); ok {
		taskReturn = robot.TaskRetVal(ln)
	}
	return taskReturn, nil
}

// modifyOSFunctions overrides os.getenv / os.setenv / os.setlocale in Lua:
//   - "getenv": if the key is in envMap, return that string; otherwise return nil
//   - "setenv": does nothing (and logs a warning)
//   - "setlocale": does nothing (and logs a warning)
//
// We NEVER call the original os.getenv. This means real OS environment variables
// are completely inaccessible via Lua's "os" table.
func modifyOSFunctions(L *glua.LState, r robot.Robot, envMap map[string]string) {
	osVal := L.GetGlobal("os")
	if osTable, ok := osVal.(*glua.LTable); ok {
		// Replace os.getenv
		osTable.RawSetString("getenv", L.NewFunction(func(L *glua.LState) int {
			key := L.CheckString(1)
			if val, found := envMap[key]; found {
				// Found in our map => return the string
				L.Push(glua.LString(val))
				return 1
			} else {
				// Not found => return nil
				L.Push(glua.LNil)
				return 1
			}
		}))

		// Replace os.setenv
		osTable.RawSetString("setenv", L.NewFunction(func(L *glua.LState) int {
			key := L.CheckString(1)
			r.Log(robot.Warn, "lua script tried to call os.setenv; ignoring for key="+key)
			// No return value
			return 0
		}))

		// Replace os.setlocale
		osTable.RawSetString("setlocale", L.NewFunction(func(L *glua.LState) int {
			locale := L.CheckString(1) // Usually the requested locale or "" in native Lua
			r.Log(robot.Warn, "Lua script tried to call os.setlocale; ignoring for locale="+locale)
			// In native Lua, setlocale would return the previous locale or nil on failure.
			// We'll just return nil to be safe.
			L.Push(glua.LNil)
			return 1
		}))
	}
}

// registerRobotType creates a "robot" metatable. In a larger codebase,
// you can keep your base methods in robotMethods or empty if you prefer
// everything in separate files.
func registerRobotType(L *glua.LState) {
	// Start with a new table for __index, and register base methods
	mt := L.NewTypeMetatable("robot")
	// Start with a new table for __index, and register base methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), robotMethods))
}

var robotMethods = map[string]glua.LGFunction{
	// If you have base-level Robot methods, put them here.
	// e.g. "CheckAdmin", "Elevate", etc.
	// "Say" and "Reply" might already be in RegisterMessageMethods(L).
}

// helper to avoid overwriting all the functions when we add new ones
func getRobotMethodTable(L *glua.LState) *glua.LTable {
	// Retrieve the metatable associated with type "robot"
	mt := L.GetTypeMetatable("robot")
	if mt == glua.LNil {
		// If for some reason "robot" isn't defined yet, create it
		mt = L.NewTypeMetatable("robot")
		L.SetMetatable(mt, mt)
	}

	// Now get the __index field from the metatable
	idx := L.GetField(mt, "__index")

	// If it’s already a table, we can just append
	if idxTable, ok := idx.(*glua.LTable); ok {
		return idxTable
	}

	// Otherwise, create a new table and set it as the __index
	newTable := L.NewTable()
	L.SetField(mt, "__index", newTable)
	return newTable
}
