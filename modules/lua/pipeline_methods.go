package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterPipelineMethods attaches pipeline-related functions to "robot".
func RegisterPipelineMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"GetParameter": robotGetParameter,
		"SetParameter": robotSetParameter,
		"Exclusive":    robotExclusive,
		"SpawnJob":     robotSpawnJob,
		"AddTask":      robotAddTask,
		"FinalTask":    robotFinalTask,
		"FailTask":     robotFailTask,
		"AddJob":       robotAddJob,
		"AddCommand":   robotAddCommand,
		"FinalCommand": robotFinalCommand,
		"FailCommand":  robotFailCommand,
	}
	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}

// -------------------------------------------------------------------
// 1) robot:GetParameter(name) -> string
// -------------------------------------------------------------------
func robotGetParameter(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "GetParameter")
		L.Push(glua.LString(""))
		return 1
	}
	if name.Type() != glua.LTString {
		lr.r.Log(robot.Error, "GetParameter requires a string argument")
		L.Push(glua.LString(""))
		return 1
	}

	val := lr.r.GetParameter(name.String())
	L.Push(glua.LString(val))
	return 1
}

// -------------------------------------------------------------------
// 2) robot:SetParameter(name, value) -> bool
// -------------------------------------------------------------------
func robotSetParameter(L *glua.LState) int {
	ud := L.CheckUserData(1)
	nameArg := L.Get(2)
	valArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "SetParameter")
		L.Push(glua.LBool(false))
		return 1
	}

	if nameArg.Type() != glua.LTString || valArg.Type() != glua.LTString {
		lr.r.Log(robot.Error, "SetParameter requires (string, string)")
		L.Push(glua.LBool(false))
		return 1
	}

	okSet := lr.r.SetParameter(nameArg.String(), valArg.String())
	L.Push(glua.LBool(okSet))
	return 1
}

// -------------------------------------------------------------------
// 3) robot:Exclusive(tag, queueTask) -> bool
// -------------------------------------------------------------------
func robotExclusive(L *glua.LState) int {
	ud := L.CheckUserData(1)
	tagArg := L.Get(2)
	queueArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "Exclusive")
		L.Push(glua.LBool(false))
		return 1
	}

	var tag string
	if tagArg.Type() == glua.LTString {
		tag = tagArg.String()
	} else {
		tag = ""
	}

	var queue bool
	if queueArg.Type() == glua.LTBool {
		queue = bool(queueArg.(glua.LBool))
	} else {
		queue = false
	}

	success := lr.r.Exclusive(tag, queue)
	L.Push(glua.LBool(success))
	return 1
}

// -------------------------------------------------------------------
// 4) robot:SpawnJob(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func robotSpawnJob(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "SpawnJob")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}
	if name.Type() != glua.LTString {
		lr.r.Log(robot.Error, "SpawnJob requires at least (string name)")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	// Collect extra args from stack positions 3..top
	extras := parseStringArgs(L, 3)

	ret := lr.r.SpawnJob(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 5) robot:AddTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func robotAddTask(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "AddTask")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}
	if name.Type() != glua.LTString {
		lr.r.Log(robot.Error, "AddTask requires at least (string name)")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	extras := parseStringArgs(L, 3)
	ret := lr.r.AddTask(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 6) robot:FinalTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func robotFinalTask(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "FinalTask")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}
	if name.Type() != glua.LTString {
		lr.r.Log(robot.Error, "FinalTask requires at least (string name)")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	extras := parseStringArgs(L, 3)
	ret := lr.r.FinalTask(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 7) robot:FailTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func robotFailTask(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "FailTask")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}
	if name.Type() != glua.LTString {
		lr.r.Log(robot.Error, "FailTask requires at least (string name)")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	extras := parseStringArgs(L, 3)
	ret := lr.r.FailTask(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 8) robot:AddJob(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func robotAddJob(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "AddJob")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}
	if name.Type() != glua.LTString {
		lr.r.Log(robot.Error, "AddJob requires at least (string name)")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	extras := parseStringArgs(L, 3)
	ret := lr.r.AddJob(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 9) robot:AddCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func robotAddCommand(L *glua.LState) int {
	ud := L.CheckUserData(1)
	pluginArg := L.Get(2)
	cmdArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "AddCommand")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	if pluginArg.Type() != glua.LTString || cmdArg.Type() != glua.LTString {
		lr.r.Log(robot.Error, "AddCommand requires (plugin, command) as strings")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	ret := lr.r.AddCommand(pluginArg.String(), cmdArg.String())
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 10) robot:FinalCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func robotFinalCommand(L *glua.LState) int {
	ud := L.CheckUserData(1)
	pluginArg := L.Get(2)
	cmdArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "FinalCommand")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	if pluginArg.Type() != glua.LTString || cmdArg.Type() != glua.LTString {
		lr.r.Log(robot.Error, "FinalCommand requires (plugin, command) as strings")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	ret := lr.r.FinalCommand(pluginArg.String(), cmdArg.String())
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 11) robot:FailCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func robotFailCommand(L *glua.LState) int {
	ud := L.CheckUserData(1)
	pluginArg := L.Get(2)
	cmdArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		logErr(lr, "FailCommand")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	if pluginArg.Type() != glua.LTString || cmdArg.Type() != glua.LTString {
		lr.r.Log(robot.Error, "FailCommand requires (plugin, command) as strings")
		L.Push(glua.LNumber(robot.Fail))
		return 1
	}

	ret := lr.r.FailCommand(pluginArg.String(), cmdArg.String())
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// Helper function to collect remaining stack arguments as strings
// from index "start" to the top. Non-string arguments are ignored.
// -------------------------------------------------------------------
func parseStringArgs(L *glua.LState, start int) []string {
	var args []string
	top := L.GetTop()
	for i := start; i <= top; i++ {
		val := L.Get(i)
		if val.Type() == glua.LTString {
			args = append(args, val.String())
		} else {
			// Optionally log or skip
			// Could do: lr.r.Log(robot.Error, "AddTask ignoring non-string argument")
		}
	}
	return args
}
