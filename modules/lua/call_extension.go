package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// luaRobot holds a reference to the robot.Robot interface, so we can call
// methods on the underlying Gopherbot Robot from Lua.
type luaRobot struct {
	r robot.Robot
}

// CallExtension loads and executes a Lua script at taskPath,
// passing in the robot.Robot, privileged mode, and the string arguments.
func CallExtension(taskPath, taskName string, r robot.Robot, privileged bool, args []string) (robot.TaskRetVal, error) {
	// 1. Create a new Lua state.
	L := glua.NewState()
	defer L.Close()

	// 2. Optionally load only certain libraries if privileged. For now, load all.
	L.OpenLibs()

	// 3. Register the "robot" type and base methods.
	registerRobotType(L)

	// 4. Now register additional method sets (message methods, logging, etc.)
	//    Just call them here (or do it inside registerRobotType, whichever).
	RegisterMessageMethods(L)
	// Example: RegisterLogMethods(L)

	// 5. Create a userdata object and assign the actual robot to it.
	robotUD := L.NewUserData()
	robotUD.Value = &luaRobot{r: r}
	L.SetMetatable(robotUD, L.GetTypeMetatable("robot"))
	L.SetGlobal("robot", robotUD)

	// 6. Register constants (RetVal, TaskRetVal, LogLevel, etc.).
	registerConstants(L)

	// 7. Provide the args table
	argsTable := L.CreateTable(len(args), 0)
	for i, a := range args {
		argsTable.RawSetInt(i+1, glua.LString(a))
	}
	L.SetGlobal("args", argsTable)

	// 8. Compile and run the Lua file.
	if err := L.DoFile(taskPath); err != nil {
		r.Log(robot.Error, fmt.Sprintf("Lua error in script '%s': %v", taskName, err))
		return robot.MechanismFail, err
	}

	// 9. Gather the final return value (defaults to robot.Normal).
	retVal := L.Get(-1) // top of the stack
	L.Pop(1)

	var taskReturn robot.TaskRetVal = robot.Normal
	if ln, ok := retVal.(glua.LNumber); ok {
		taskReturn = robot.TaskRetVal(ln)
	}
	return taskReturn, nil
}

// registerRobotType creates a "robot" metatable with base methods.
func registerRobotType(L *glua.LState) {
	mt := L.NewTypeMetatable("robot")
	// Start with a new table for __index, and register base methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), robotMethods))
}

// robotMethods are the base set of methods we consider “core” to Robot.
// Additional sets (like SendMessage, Log, etc.) get added separately.
var robotMethods = map[string]glua.LGFunction{
	// For example, if you have an "Elevate" or "CheckAdmin" method, put it here.
	// "Say" or "Reply" can go here or in a separate file, depending on preference.
}

// getRobotMethodTable fetches the existing __index table for the "robot" type,
// so we can add more methods without overwriting the entire table.
func getRobotMethodTable(L *glua.LState) *glua.LTable {
	mt := L.GetTypeMetatable("robot") // get metatable for "robot"
	idx := L.GetField(mt, "__index")  // get the current __index
	if idxTable, ok := idx.(*glua.LTable); ok {
		return idxTable
	}
	// If it wasn't a table, create a new one just in case.
	newTable := L.NewTable()
	L.SetField(mt, "__index", newTable)
	return newTable
}
