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
	luaRobot
	L *glua.LState
}

// CallExtension loads and executes a Lua script:
//   - taskPath, taskName - the path to script and its name
//   - pkgPath - directories the script should search for requires
//   - env - env vars normally passed to external scripts, has thread info
//   - r: the robot.Robot
//   - args: the script arguments
func CallExtension(taskPath, taskName string, pkgPath []string, env map[string]string, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	L := glua.NewState()
	defer L.Close()

	// Initialize the luaRobot with fields from envHash
	initialFields, err := initializeFields(env)
	if err != nil {
		return robot.MechanismFail, err
	}

	lr := luaRobot{
		r:      r,
		fields: initialFields,
	}

	lctx := luaContext{
		luaRobot: lr,
		L:        L,
	}

	// Open standard libraries
	L.OpenLibs()

	// Modify OS functions to replace os.setenv and os.setlocale with no-ops
	modifyOSFunctions(L, r)

	// Register the "robot" type and its metamethods (only New method)
	registerRobotType(&lctx)

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
	robotUD := newLuaRobot(L, r, initialFields)
	L.SetGlobal("robot", robotUD)

	// Provide the script arguments as a standard Lua "arg" table
	argsTable := L.CreateTable(len(args)+1, 0)
	argsTable.RawSetInt(0, glua.LString(taskName))
	for i, a := range args {
		argsTable.RawSetInt(i+1, glua.LString(a))
	}
	L.SetGlobal("arg", argsTable)

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

// initializeFields initializes the robot fields from envHash.
func initializeFields(env map[string]string) (map[string]interface{}, error) {
	fields := make(map[string]interface{})

	// List of predefined string fields
	stringFields := []string{
		"channel",
		"channel_id",
		"message_id",
		"thread_id",
		"user",
		"user_id",
		"plugin_id",
		"protocol",
		"brain",
		"format",
	}

	for _, key := range stringFields {
		envKey := "GOPHER_" + strings.ToUpper(key)
		if val, exists := env[envKey]; exists {
			fields[key] = val
		} else {
			fields[key] = "" // Default to empty string if not set
		}
	}

	// Handle threaded_message as boolean
	threadedVal, exists := env["GOPHER_THREADED_MESSAGE"]
	if exists && strings.ToLower(threadedVal) == "true" {
		fields["threaded_message"] = true
	} else {
		fields["threaded_message"] = false
	}

	return fields, nil
}

// updatePkgPath appends additional paths to Lua's package.path
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

// modifyOSFunctions overrides os.setenv and os.setlocale in Lua to prevent modifications
func modifyOSFunctions(L *glua.LState, r robot.Robot) {
	osVal := L.GetGlobal("os")
	if osTable, ok := osVal.(*glua.LTable); ok {
		// Replace os.setenv
		osTable.RawSetString("setenv", L.NewFunction(func(L *glua.LState) int {
			key := L.CheckString(1)
			r.Log(robot.Warn, "Lua script tried to call os.setenv; ignoring for key="+key)
			// No return value
			return 0
		}))

		// Replace os.setlocale
		osTable.RawSetString("setlocale", L.NewFunction(func(L *glua.LState) int {
			locale := L.CheckString(1)
			r.Log(robot.Warn, "Lua script tried to call os.setlocale; ignoring for locale="+locale)
			// Return nil to mimic Lua's behavior
			L.Push(glua.LNil)
			return 1
		}))
	}
}

// registerRobotType sets up the metatable for the primary robot userdata with only the __index metamethod
func registerRobotType(lctx *luaContext) {
	L := lctx.L

	// Create a new metatable for "robot"
	mt := L.NewTypeMetatable("robot")

	// Set the __index metamethod to access methods (only "New")
	L.SetField(mt, "__index", L.NewFunction(lctx.robotIndex))

	// Register the New method
	methods := map[string]glua.LGFunction{
		"New": lctx.robotNew,
	}
	methodTable := L.NewTable()
	L.SetFuncs(methodTable, methods)
	L.SetField(mt, "methods", methodTable)
}

// robotIndex handles the __index metamethod for primary robot userdata
func (lctx *luaContext) robotIndex(L *glua.LState) int {
	// The userdata is at index 1
	L.CheckUserData(1) // Ensure it's userdata

	key := L.CheckString(2)

	// Retrieve the metatable associated with "robot"
	mt := L.GetTypeMetatable("robot")
	if mt == glua.LNil {
		lctx.logRobotErr("__index")
		L.RaiseError("robot metatable not found")
		return 0
	}

	// Get the "methods" table from the metatable
	methods := L.GetField(mt, "methods")
	tbl, ok := methods.(*glua.LTable)
	if !ok {
		lctx.logRobotErr("__index")
		L.RaiseError("methods table not found in robot metatable")
		return 0
	}

	// Get the method named 'key' from the "methods" table
	method := tbl.RawGetString(key)
	if method != glua.LNil {
		L.Push(method)
		return 1
	}

	// If method not found, return nil
	L.Push(glua.LNil)
	return 1
}

// robotNew creates a new bot userdata instance with fields copied from envHash
func (lctx *luaContext) robotNew(L *glua.LState) int {
	// Retrieve the primary robot userdata
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logRobotErr("New")
		L.RaiseError("Invalid robot userdata for New()")
		return 0
	}

	// Initialize fields from envHash
	newFields, err := initializeFields(lctx.fieldsFromEnv())
	if err != nil {
		L.RaiseError("Failed to initialize new robot: %v", err)
		return 0
	}

	// Create a new bot userdata
	newUD := newLuaBot(L, lr.r, newFields)
	L.Push(newUD)
	return 1
}

// fieldsFromEnv retrieves the original envHash used for initializing fields
func (lr *luaRobot) fieldsFromEnv() map[string]string {
	env := make(map[string]string)
	for key, value := range lr.fields {
		switch v := value.(type) {
		case string:
			env["GOPHER_"+strings.ToUpper(key)] = v
		case bool:
			if v {
				env["GOPHER_"+strings.ToUpper(key)] = "true"
			} else {
				env["GOPHER_"+strings.ToUpper(key)] = "false"
			}
		}
	}
	return env
}

// newLuaRobot creates a new Lua userdata for the primary robot with only the New method.
func newLuaRobot(L *glua.LState, r robot.Robot, fields map[string]interface{}) *glua.LUserData {
	newUD := L.NewUserData()
	newUD.Value = &luaRobot{r: r, fields: fields}
	// Set the metatable for "robot"
	L.SetMetatable(newUD, L.GetTypeMetatable("robot"))
	return newUD
}

// logRobotErr logs an error specific to the primary robot userdata.
func (lctx *luaContext) logRobotErr(caller string) {
	if lctx.r != nil {
		lctx.r.Log(robot.Error, fmt.Sprintf("%s called with invalid robot userdata", caller))
	} else {
		fmt.Printf("[ERR] %s called but robot is nil\n", caller)
	}
}

// pushFail is a helper to push a failure code onto the Lua stack
func pushFail(L *glua.LState) int {
	L.Push(glua.LNumber(robot.Failed))
	return 1
}
