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
// It returns a robot.TaskRetVal and an error (if something goes wrong
// in the Lua execution).
func CallExtension(taskPath, taskName string, r robot.Robot, privileged bool, args []string) (robot.TaskRetVal, error) {
	// 1. Create a new Lua state.
	L := glua.NewState()
	defer L.Close()

	// 2. Optionally load only certain libraries based on privileged or unprivileged.
	//    For now, load all standard libraries for demonstration.
	L.OpenLibs()

	// 3. Register the Robot userdata type and set its methods.
	registerRobotType(L)

	// 4. Create a userdata object and assign the actual robot to it.
	robotUD := L.NewUserData()
	robotUD.Value = &luaRobot{r: r}
	L.SetMetatable(robotUD, L.GetTypeMetatable("robot"))
	// Make it globally accessible as "robot".
	L.SetGlobal("robot", robotUD)

	// 5. Register constants for robot.RetVal (method returns) and robot.TaskRetVal (script returns).
	registerConstants(L)

	// 6. Set the global "args" table so Lua can access the arguments.
	argsTable := L.CreateTable(len(args), 0)
	for i, a := range args {
		argsTable.RawSetInt(i+1, glua.LString(a))
	}
	L.SetGlobal("args", argsTable)

	// 7. Compile and run the Lua file.
	if err := L.DoFile(taskPath); err != nil {
		// A compile or runtime error in Lua => MechanismFail
		r.Log(robot.Error, fmt.Sprintf("Lua error in script '%s': %v", taskName, err))
		return robot.MechanismFail, err
	}

	// 8. After successful script execution, check what the script returned.
	//    By convention, we expect exactly one return value from the script.
	//    If the script didn’t explicitly return anything, that’s fine — we’ll default to Normal.
	retVal := L.Get(-1) // top of the stack
	L.Pop(1)            // pop it off

	// Convert that Lua value to an int => robot.TaskRetVal
	var taskReturn robot.TaskRetVal = robot.Normal
	if ln, ok := retVal.(glua.LNumber); ok {
		taskReturn = robot.TaskRetVal(ln) // e.g. 0 => Normal, 1 => Fail, etc.
	}

	// 9. Return the final task return value.
	return taskReturn, nil
}

// registerRobotType sets up a metatable for "robot" userdata and binds Go
// methods (like :Say) to be callable from Lua.
func registerRobotType(L *glua.LState) {
	// Create a metatable for type "robot".
	mt := L.NewTypeMetatable("robot")
	// Set __index to a table of methods.
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), robotMethods))
}

// robotMethods binds each robot.* Lua method name to the corresponding Go function.
var robotMethods = map[string]glua.LGFunction{
	"Say": robotSay,
	// In the future, add more: "Reply", "Log", etc.
}

// robotSay implements the robot:Say("some message") call.
func robotSay(L *glua.LState) int {
	// 1st argument is the userdata (self), 2nd is the message string.
	ud := L.CheckUserData(1)
	msg := L.CheckString(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		// RaiseError throws a Lua runtime error.
		L.RaiseError("invalid robot userdata")
		return 0
	}

	// Call the actual Go robot’s Say method.
	ret := lr.r.Say(msg)
	// Push the RetVal (int) onto the Lua stack.
	L.Push(glua.LNumber(ret))
	// Return 1 value back to Lua (the ret code).
	return 1
}

// registerConstants registers a minimal set of robot.RetVal and robot.TaskRetVal constants
// into the global Lua environment.
func registerConstants(L *glua.LState) {
	// --------------------------------------------------------------------
	// Robot method return codes (RetVal) => "retXYZ"
	// e.g., retOk, retFailedMessageSend, etc.
	// --------------------------------------------------------------------
	L.SetGlobal("retOk", glua.LNumber(robot.Ok))
	L.SetGlobal("retUserNotFound", glua.LNumber(robot.UserNotFound))
	L.SetGlobal("retChannelNotFound", glua.LNumber(robot.ChannelNotFound))
	L.SetGlobal("retFailedMessageSend", glua.LNumber(robot.FailedMessageSend))
	// ... Add more as needed ...
	// L.SetGlobal("retInterrupted", glua.LNumber(robot.Interrupted))
	// L.SetGlobal("retReplyNotMatched", glua.LNumber(robot.ReplyNotMatched))
	// etc.

	// --------------------------------------------------------------------
	// Task return codes (TaskRetVal) => "taskXYZ"
	// e.g., taskNormal, taskFail, taskMechanismFail, etc.
	// --------------------------------------------------------------------
	L.SetGlobal("taskNormal", glua.LNumber(robot.Normal)) // 0
	L.SetGlobal("taskFail", glua.LNumber(robot.Fail))     // 1
	L.SetGlobal("taskMechanismFail", glua.LNumber(robot.MechanismFail))
	L.SetGlobal("taskConfigurationError", glua.LNumber(robot.ConfigurationError))
	L.SetGlobal("taskSuccess", glua.LNumber(robot.Success)) // 7
	// ... Add more as needed ...
}
