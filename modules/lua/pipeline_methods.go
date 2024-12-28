package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterPipelineMethods attaches pipeline-related functions to the bot metatable:
//
//	bot:GetParameter(name) -> string
//	bot:SetParameter(name, value) -> bool
//	bot:Exclusive(tag, queueTask) -> bool
//	bot:SpawnJob(name, arg1, arg2, ...)
//	bot:AddTask(name, arg1, arg2, ...)
//	bot:FinalTask(name, arg1, arg2, ...)
//	bot:FailTask(name, arg1, arg2, ...)
//	bot:AddJob(name, arg1, arg2, ...)
//	bot:AddCommand(pluginName, command) -> RetVal
//	bot:FinalCommand(pluginName, command) -> RetVal
//	bot:FailCommand(pluginName, command) -> RetVal
func (lctx luaContext) RegisterPipelineMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"GetParameter": lctx.botGetParameter,
		"SetParameter": lctx.botSetParameter,
		"Exclusive":    lctx.botExclusive,
		"SpawnJob":     lctx.botSpawnJob,
		"AddTask":      lctx.botAddTask,
		"FinalTask":    lctx.botFinalTask,
		"FailTask":     lctx.botFailTask,
		"AddJob":       lctx.botAddJob,
		"AddCommand":   lctx.botAddCommand,
		"FinalCommand": lctx.botFinalCommand,
		"FailCommand":  lctx.botFailCommand,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// -------------------------------------------------------------------
// 1) bot:GetParameter(name) -> string
// -------------------------------------------------------------------
func (lctx luaContext) botGetParameter(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("GetParameter")
		return pushFail(L)
	}

	if name.Type() != glua.LTString {
		lctx.Log(robot.Error, "GetParameter requires a string argument")
		return pushFail(L)
	}

	val := lr.r.GetParameter(name.String())
	L.Push(glua.LString(val))
	return 1
}

// -------------------------------------------------------------------
// 2) bot:SetParameter(name, value) -> bool
// -------------------------------------------------------------------
func (lctx luaContext) botSetParameter(L *glua.LState) int {
	ud := L.CheckUserData(1)
	nameArg := L.Get(2)
	valArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("SetParameter")
		return pushFail(L)
	}

	if nameArg.Type() != glua.LTString || valArg.Type() != glua.LTString {
		lctx.Log(robot.Error, "SetParameter requires (string, string)")
		return pushFail(L)
	}

	okSet := lr.r.SetParameter(nameArg.String(), valArg.String())
	L.Push(glua.LBool(okSet))
	return 1
}

// -------------------------------------------------------------------
// 3) bot:Exclusive(tag, queueTask) -> bool
// -------------------------------------------------------------------
func (lctx luaContext) botExclusive(L *glua.LState) int {
	ud := L.CheckUserData(1)
	tagArg := L.Get(2)
	queueArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("Exclusive")
		return pushFail(L)
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
// 4) bot:SpawnJob(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx luaContext) botSpawnJob(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("SpawnJob")
		return pushFail(L)
	}

	if name.Type() != glua.LTString {
		lctx.Log(robot.Error, "SpawnJob requires at least (string name)")
		return pushFail(L)
	}

	// Collect extra args from stack positions 3..top
	extras := parseStringArgs(L, 3)

	ret := lr.r.SpawnJob(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 5) bot:AddTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx luaContext) botAddTask(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("AddTask")
		return pushFail(L)
	}

	if name.Type() != glua.LTString {
		lctx.Log(robot.Error, "AddTask requires at least (string name)")
		return pushFail(L)
	}

	extras := parseStringArgs(L, 3)
	ret := lr.r.AddTask(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 6) bot:FinalTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx luaContext) botFinalTask(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("FinalTask")
		return pushFail(L)
	}

	if name.Type() != glua.LTString {
		lctx.Log(robot.Error, "FinalTask requires at least (string name)")
		return pushFail(L)
	}

	extras := parseStringArgs(L, 3)
	ret := lr.r.FinalTask(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 7) bot:FailTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx luaContext) botFailTask(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("FailTask")
		return pushFail(L)
	}

	if name.Type() != glua.LTString {
		lctx.Log(robot.Error, "FailTask requires at least (string name)")
		return pushFail(L)
	}

	extras := parseStringArgs(L, 3)
	ret := lr.r.FailTask(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 8) bot:AddJob(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx luaContext) botAddJob(L *glua.LState) int {
	ud := L.CheckUserData(1)
	name := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("AddJob")
		return pushFail(L)
	}

	if name.Type() != glua.LTString {
		lctx.Log(robot.Error, "AddJob requires at least (string name)")
		return pushFail(L)
	}

	extras := parseStringArgs(L, 3)
	ret := lr.r.AddJob(name.String(), extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 9) bot:AddCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func (lctx luaContext) botAddCommand(L *glua.LState) int {
	ud := L.CheckUserData(1)
	pluginArg := L.Get(2)
	cmdArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("AddCommand")
		return pushFail(L)
	}

	if pluginArg.Type() != glua.LTString || cmdArg.Type() != glua.LTString {
		lctx.Log(robot.Error, "AddCommand requires (plugin, command) as strings")
		return pushFail(L)
	}

	ret := lr.r.AddCommand(pluginArg.String(), cmdArg.String())
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 10) bot:FinalCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func (lctx luaContext) botFinalCommand(L *glua.LState) int {
	ud := L.CheckUserData(1)
	pluginArg := L.Get(2)
	cmdArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("FinalCommand")
		return pushFail(L)
	}

	if pluginArg.Type() != glua.LTString || cmdArg.Type() != glua.LTString {
		lctx.Log(robot.Error, "FinalCommand requires (plugin, command) as strings")
		return pushFail(L)
	}

	ret := lr.r.FinalCommand(pluginArg.String(), cmdArg.String())
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 11) bot:FailCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func (lctx luaContext) botFailCommand(L *glua.LState) int {
	ud := L.CheckUserData(1)
	pluginArg := L.Get(2)
	cmdArg := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("FailCommand")
		return pushFail(L)
	}

	if pluginArg.Type() != glua.LTString || cmdArg.Type() != glua.LTString {
		lctx.Log(robot.Error, "FailCommand requires (plugin, command) as strings")
		return pushFail(L)
	}

	ret := lr.r.FailCommand(pluginArg.String(), cmdArg.String())
	L.Push(glua.LNumber(ret))
	return 1
}
