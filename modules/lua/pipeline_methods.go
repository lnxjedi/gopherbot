package lua

import (
	glua "github.com/yuin/gopher-lua"
)

// RegisterPipelineMethods attaches pipeline-related functions to the bot metatable:
//
//	bot:GetParameter(name) -> string
//	bot:SetParameter(name, value) -> bool
//	bot:Subscribe() -> bool
//	bot:Unsubscribe() -> bool
//	bot:Exclusive(tag, queueTask) -> bool
//	bot:SpawnJob(name, arg1, arg2, ...)
//	bot:AddTask(name, arg1, arg2, ...)
//	bot:FinalTask(name, arg1, arg2, ...)
//	bot:FailTask(name, arg1, arg2, ...)
//	bot:AddJob(name, arg1, arg2, ...)
//	bot:AddCommand(pluginName, command) -> RetVal
//	bot:FinalCommand(pluginName, command) -> RetVal
//	bot:FailCommand(pluginName, command) -> RetVal
func (lctx *luaContext) RegisterPipelineMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"GetParameter": lctx.botGetParameter,
		"SetParameter": lctx.botSetParameter,
		"Subscribe":    lctx.botSubscribe,
		"Unsubscribe":  lctx.botUnsubscribe,
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
func (lctx *luaContext) botGetParameter(L *glua.LState) int {
	r := lctx.getRobot(L, "GetParameter")
	name := L.CheckString(2)

	if name == "" {
		L.RaiseError("GetParameter: name must not be empty")
		return 0
	}

	val := r.GetParameter(name)
	L.Push(glua.LString(val))
	return 1
}

// -------------------------------------------------------------------
// 2) bot:SetParameter(name, value) -> bool
// -------------------------------------------------------------------
func (lctx *luaContext) botSetParameter(L *glua.LState) int {
	r := lctx.getRobot(L, "SetParameter")
	name := L.CheckString(2)
	value := L.CheckString(3)

	if name == "" {
		L.RaiseError("SetParameter: name must not be empty")
		return 0
	}

	okSet := r.SetParameter(name, value)
	L.Push(glua.LBool(okSet))
	return 1
}

// -------------------------------------------------------------------
// 3) bot:Subscribe() -> bool
// -------------------------------------------------------------------
func (lctx *luaContext) botSubscribe(L *glua.LState) int {
	r := lctx.getRobot(L, "Subscribe")
	success := r.Subscribe()
	L.Push(glua.LBool(success))
	return 1
}

// -------------------------------------------------------------------
// 4) bot:Unsubscribe() -> bool
// -------------------------------------------------------------------
func (lctx *luaContext) botUnsubscribe(L *glua.LState) int {
	r := lctx.getRobot(L, "Unsubscribe")
	success := r.Unsubscribe()
	L.Push(glua.LBool(success))
	return 1
}

// -------------------------------------------------------------------
// 5) bot:Exclusive(tag, queueTask) -> bool
// -------------------------------------------------------------------
func (lctx *luaContext) botExclusive(L *glua.LState) int {
	r := lctx.getRobot(L, "Exclusive")
	tag := L.CheckString(2)
	queue := L.CheckBool(3)

	if tag == "" {
		L.RaiseError("Exclusive: tag must not be empty")
		return 0
	}

	success := r.Exclusive(tag, queue)
	L.Push(glua.LBool(success))
	return 1
}

// -------------------------------------------------------------------
// 6) bot:SpawnJob(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx *luaContext) botSpawnJob(L *glua.LState) int {
	r := lctx.getRobot(L, "SpawnJob")
	name := L.CheckString(2)

	if name == "" {
		L.RaiseError("SpawnJob: name must not be empty")
		return 0
	}

	extras := parseStringArgs(L, 3)
	ret := r.SpawnJob(name, extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 7) bot:AddTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx *luaContext) botAddTask(L *glua.LState) int {
	r := lctx.getRobot(L, "AddTask")
	name := L.CheckString(2)

	if name == "" {
		L.RaiseError("AddTask: name must not be empty")
		return 0
	}

	extras := parseStringArgs(L, 3)
	ret := r.AddTask(name, extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 8) bot:FinalTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx *luaContext) botFinalTask(L *glua.LState) int {
	r := lctx.getRobot(L, "FinalTask")
	name := L.CheckString(2)

	if name == "" {
		L.RaiseError("FinalTask: name must not be empty")
		return 0
	}

	extras := parseStringArgs(L, 3)
	ret := r.FinalTask(name, extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 9) bot:FailTask(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx *luaContext) botFailTask(L *glua.LState) int {
	r := lctx.getRobot(L, "FailTask")
	name := L.CheckString(2)

	if name == "" {
		L.RaiseError("FailTask: name must not be empty")
		return 0
	}

	extras := parseStringArgs(L, 3)
	ret := r.FailTask(name, extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 10) bot:AddJob(name, arg1, arg2, ... argN) -> RetVal
// -------------------------------------------------------------------
func (lctx *luaContext) botAddJob(L *glua.LState) int {
	r := lctx.getRobot(L, "AddJob")
	name := L.CheckString(2)

	if name == "" {
		L.RaiseError("AddJob: name must not be empty")
		return 0
	}

	extras := parseStringArgs(L, 3)
	ret := r.AddJob(name, extras...)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 11) bot:AddCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func (lctx *luaContext) botAddCommand(L *glua.LState) int {
	r := lctx.getRobot(L, "AddCommand")
	pluginName := L.CheckString(2)
	command := L.CheckString(3)

	if pluginName == "" {
		L.RaiseError("AddCommand: pluginName must not be empty")
		return 0
	}
	if command == "" {
		L.RaiseError("AddCommand: command must not be empty")
		return 0
	}

	ret := r.AddCommand(pluginName, command)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 12) bot:FinalCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func (lctx *luaContext) botFinalCommand(L *glua.LState) int {
	r := lctx.getRobot(L, "FinalCommand")
	pluginName := L.CheckString(2)
	command := L.CheckString(3)

	if pluginName == "" {
		L.RaiseError("FinalCommand: pluginName must not be empty")
		return 0
	}
	if command == "" {
		L.RaiseError("FinalCommand: command must not be empty")
		return 0
	}

	ret := r.FinalCommand(pluginName, command)
	L.Push(glua.LNumber(ret))
	return 1
}

// -------------------------------------------------------------------
// 13) bot:FailCommand(pluginName, command) -> RetVal
// -------------------------------------------------------------------
func (lctx *luaContext) botFailCommand(L *glua.LState) int {
	r := lctx.getRobot(L, "FailCommand")
	pluginName := L.CheckString(2)
	command := L.CheckString(3)

	if pluginName == "" {
		L.RaiseError("FailCommand: pluginName must not be empty")
		return 0
	}
	if command == "" {
		L.RaiseError("FailCommand: command must not be empty")
		return 0
	}

	ret := r.FailCommand(pluginName, command)
	L.Push(glua.LNumber(ret))
	return 1
}
