package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
	lua "github.com/yuin/gopher-lua"
)

// luaRobot holds a reference to the robot.Robot interface, so we can
// call methods on the underlying Gopherbot Robot from Lua.
type luaRobot struct {
	r robot.Robot
}

// CallExtension loads and executes a Lua script at taskPath, passing in:
//   - env: a map of environment-like variables to expose in Lua as a read-only table
//   - r: the robot.Robot
//   - privileged: if true, we don't remove any functions; if false, we remove os.getenv, os.setenv
//   - args: the script arguments
func CallExtension(taskPath, taskName string, env map[string]string, r robot.Robot, privileged bool, args []string) (robot.TaskRetVal, error) {
	// 1. Create a new Lua state.
	L := glua.NewState()
	defer L.Close()

	// 2. Load all libraries, then selectively remove os.getenv and os.setenv if unprivileged.
	// L.OpenLibs()

	modifyEnvFunctions(L, r, env)

	// 3. Register the "robot" type and base methods.
	registerRobotType(L)

	// 3a. Register additional sets (e.g., message methods):
	RegisterMessageMethods(L)
	// If you have other method groups (Log, Memory, etc.), register them here too.
	// RegisterLogMethods(L)
	// RegisterMemoryMethods(L)

	// 4. Create the robot userdata object and set it as "robot".
	robotUD := L.NewUserData()
	robotUD.Value = &luaRobot{r: r}
	L.SetMetatable(robotUD, L.GetTypeMetatable("robot"))
	L.SetGlobal("robot", robotUD)

	// 5. Register constants (RetVal, TaskRetVal, etc.).
	registerConstants(L)

	// 6. Provide the script arguments as a Lua table "args".
	argsTable := L.CreateTable(len(args), 0)
	for i, a := range args {
		argsTable.RawSetInt(i+1, glua.LString(a))
	}
	L.SetGlobal("args", argsTable)

	// 7. Create an "env" table from the provided env map and set it global.
	envTable := L.CreateTable(0, len(env))
	for k, v := range env {
		envTable.RawSetString(k, glua.LString(v))
	}
	L.SetGlobal("env", envTable)

	// 8. Compile and run the Lua file.
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

// modifyEnvFunctions overrides os.getenv / os.setenv in Lua:
//   - "getenv": if the key is in envMap, return that string; otherwise return nil
//   - "setenv": does nothing (and logs a warning).
//
// We NEVER call the original os.getenv. This means real OS environment variables
// are completely inaccessible via Lua's "os" table.
func modifyEnvFunctions(L *glua.LState, r robot.Robot, envMap map[string]string) {
	osVal := L.GetGlobal("os")
	if osTable, ok := osVal.(*glua.LTable); ok {
		// Replace os.getenv
		osTable.RawSetString("getenv", L.NewFunction(func(L *lua.LState) int {
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
		osTable.RawSetString("setenv", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)
			r.Log(robot.Warn, "lua script tried to call os.setenv; ignoring for key="+key)
			// No return value
			return 0
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
