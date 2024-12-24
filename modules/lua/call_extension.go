package lua

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// luaRobot holds a reference to the robot.Robot interface, so we can
// call methods on the underlying Gopherbot Robot from Lua.
type luaRobot struct {
	r   robot.Robot
	env map[string]string
}

// CallExtension loads and executes a Lua script:
//   - taskPath, taskName - the path to script and it's name
//   - pkgPath - directories the script should search for requires
//   - env - env vars normally passed to external scripts, has thread info
//   - r: the robot.Robot
//   - args: the script arguments
func CallExtension(taskPath, taskName string, pkgPath []string, env map[string]string, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	L := glua.NewState()
	defer L.Close()

	// This is done automatically unless the SkipOpenLibs option is passed
	// L.OpenLibs()

	// Modify OS functions to replace os.setenv and os.setlocale with no-ops
	modifyOSFunctions(L, r, env)

	// Register the "robot" type and any base methods
	registerRobotType(L)

	// Register additional method sets
	RegisterMessageMethods(L)
	RegisterRobotModifiers(L)
	RegisterLongTermMemoryMethods(L)
	RegisterShortTermMemoryMethods(L)
	RegisterConfigMethod(L)
	RegisterUtilMethods(L)
	RegisterAttributeMethods(L)
	RegisterPromptingMethods(L)
	RegisterPipelineMethods(L)

	// Create the robot userdata object and set it as "robot"
	robotUD := L.NewUserData()
	robotUD.Value = &luaRobot{r: r, env: env}
	L.SetMetatable(robotUD, L.GetTypeMetatable("robot"))
	L.SetGlobal("robot", robotUD)

	// Provide the script arguments as a standard Lua "arg" table
	argsTable := L.CreateTable(len(args)+1, 0)
	argsTable.RawSetInt(0, glua.LString(taskName))
	for i, a := range args {
		argsTable.RawSetInt(i+1, glua.LString(a))
	}
	L.SetGlobal("arg", argsTable)

	// **Update package.path with additional directories and Lua patterns**
	ret, err := updatePkgPath(L, r, pkgPath)
	if err != nil {
		return ret, err
	}

	// Compile and run the Lua file
	if err := L.DoFile(taskPath); err != nil {
		r.Log(robot.Error, fmt.Sprintf("Lua error in script '%s': %v", taskName, err))
		return robot.MechanismFail, err
	}

	// Check the script’s return value (default to Normal)
	retVal := L.Get(-1) // top of stack
	L.Pop(1)

	var taskReturn robot.TaskRetVal = robot.Normal
	if ln, ok := retVal.(glua.LNumber); ok {
		taskReturn = robot.TaskRetVal(ln)
	}
	return taskReturn, nil
}

func updatePkgPath(L *glua.LState, r robot.Robot, pkgPath []string) (robot.TaskRetVal, error) {
	var additionalPaths []string
	for _, dir := range pkgPath {
		// Ensure no trailing slash
		dir = strings.TrimRight(dir, "/")

		// Append Lua patterns
		additionalPaths = append(additionalPaths, fmt.Sprintf("%s/?.lua", dir))
		additionalPaths = append(additionalPaths, fmt.Sprintf("%s/?/init.lua", dir))
	}

	// Join the additional paths with semicolons
	additionalPathsStr := strings.Join(additionalPaths, ";")

	// Lua code to append the additional paths to package.path
	luaPathUpdate := fmt.Sprintf(`package.path = package.path .. ";%s"`, additionalPathsStr)

	// Execute the Lua code to update package.path
	if err := L.DoString(luaPathUpdate); err != nil {
		if r != nil {
			r.Log(robot.Error, fmt.Sprintf("Failed to update package.path: %v", err))
		} else {
			fmt.Println("failed to update package.path in modules/lua")
		}
		return robot.MechanismFail, err
	}
	return robot.Normal, nil
}

// logErr logs an error if lr or lr.r is valid, otherwise prints to stdout.
func logErr(lr *luaRobot, caller string) {
	if lr != nil && lr.r != nil {
		lr.r.Log(robot.Error, fmt.Sprintf("%s called with invalid robot userdata", caller))
	} else {
		fmt.Printf("[ERR] %s called but robot is nil\n", caller)
	}
}

// modifyOSFunctions overrides os.setenv / os.setlocale in Lua:
//   - "setenv": does nothing (and logs a warning)
//   - "setlocale": does nothing (and logs a warning)
func modifyOSFunctions(L *glua.LState, r robot.Robot, envMap map[string]string) {
	osVal := L.GetGlobal("os")
	if osTable, ok := osVal.(*glua.LTable); ok {
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
