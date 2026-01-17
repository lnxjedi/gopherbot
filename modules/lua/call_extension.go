// call_extension.go
package lua

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// luaRobot encapsulates the Go robot.Robot interface and its fields.
type luaRobot struct {
	r      robot.Robot
	fields map[string]interface{}
}

// luaContext holds a reference to the robot.Robot interface and the Lua state.
type luaContext struct {
	robot.Logger
	L   *glua.LState
	bot map[string]string
}

// CallExtension loads and executes a Lua script:
//   - taskPath, taskName - the path to script and its name
//   - pkgPath - directories the script should search for requires
//   - env - env vars normally passed to external scripts, has thread info
//   - r: the robot.Robot
//   - args: the script arguments
func CallExtension(execPath, taskPath, taskName string, pkgPath []string, logger robot.Logger,
	bot map[string]string, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	L := glua.NewState()
	defer L.Close()

	// Add the simple http interface
	addHttpHandler(L)

	lctx := luaContext{
		logger,
		L, // LState
		bot,
	}

	// Create a Lua arg table
	addArgTable(L, execPath, taskPath, args...)

	// Modify OS functions to replace os.setenv and os.setlocale with no-ops
	modifyOSFunctions(L, r)

	// Well, the FIRST time it's called it definitely will - but not in all the
	// component Register* functions below ...
	registerBotMetatableIfNeeded(L)

	// Register additional method sets for "bot" userdatas
	// (each function merges its methods into the "bot" metatable)
	lctx.RegisterMessageMethods(L)
	lctx.RegisterRobotModifiers(L)
	lctx.RegisterLongTermMemoryMethods(L)
	lctx.RegisterShortTermMemoryMethods(L)
	lctx.RegisterConfigMethod(L)
	lctx.RegisterUtilMethods(L)
	lctx.RegisterAttributeMethods(L)
	lctx.RegisterPromptingMethods(L)
	lctx.RegisterPipelineMethods(L)

	// Create the primary robot userdata and set it as "robot"
	robotUD := lctx.newLuaBot(L, r)
	L.SetGlobal("GBOT", robotUD)

	// Update package.path with additional directories and Lua patterns
	ret, err := updatePkgPath(L, r, pkgPath)
	if err != nil {
		return ret, err
	}

	// Compile and run the Lua file
	if err := L.DoFile(taskPath); err != nil {
		return robot.MechanismFail, fmt.Errorf("Lua error in script '%s': %w", taskName, err)
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

// addArgTable creates a Lua table named 'arg' and populates it.
// The first two entries are special:
//
//	arg[-1] is the path to the interpreter (execPath)
//	arg[0]  is the path to the script being run (taskPath)
//
// Following these are the actual command arguments.
func addArgTable(L *glua.LState, execPath, taskPath string, args ...string) {
	argTable := L.NewTable()

	// Set special entries for interpreter and script path
	L.SetTable(argTable, glua.LNumber(-1), glua.LString(execPath))
	L.SetTable(argTable, glua.LNumber(0), glua.LString(taskPath))

	// Add command arguments
	for i, arg := range args {
		L.SetTable(argTable, glua.LNumber(i+1), glua.LString(arg))
	}

	// Set the global 'arg' table
	L.SetGlobal("arg", argTable)
}

// updatePkgPath appends additional paths to Lua's package.path
func updatePkgPath(L *glua.LState, l robot.Logger, pkgPath []string) (robot.TaskRetVal, error) {
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
		l.Log(robot.Error, fmt.Sprintf("Failed to update package.path: %v", err))
		return robot.MechanismFail, err
	}
	return robot.Normal, nil
}

// modifyOSFunctions overrides os.setenv and os.setlocale in Lua to prevent modifications
func modifyOSFunctions(L *glua.LState, l robot.Logger) {
	osVal := L.GetGlobal("os")
	if osTable, ok := osVal.(*glua.LTable); ok {
		// Replace os.setenv
		osTable.RawSetString("setenv", L.NewFunction(func(L *glua.LState) int {
			key := L.CheckString(1)
			l.Log(robot.Warn, "Lua script tried to call os.setenv; ignoring for key="+key)
			// No return value
			return 0
		}))

		// Replace os.setlocale
		osTable.RawSetString("setlocale", L.NewFunction(func(L *glua.LState) int {
			locale := L.CheckString(1)
			l.Log(robot.Warn, "Lua script tried to call os.setlocale; ignoring for locale="+locale)
			// Return nil to mimic Lua's behavior
			L.Push(glua.LNil)
			return 1
		}))
	}
}
