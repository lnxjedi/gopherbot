package lua

import (
	glua "github.com/yuin/gopher-lua"
)

// RegisterShortTermMemoryMethods adds ephemeral memory methods to the bot's metatable:
//
//	bot:Remember(key, value, shared)
//	bot:RememberThread(key, value, shared)
//	bot:RememberContext(context, value)
//	bot:RememberContextThread(context, value)
//	bot:Recall(key, shared) -> string
func (lctx *luaContext) RegisterShortTermMemoryMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"Remember":              lctx.botRemember,
		"RememberThread":        lctx.botRememberThread,
		"RememberContext":       lctx.botRememberContext,
		"RememberContextThread": lctx.botRememberContextThread,
		"Recall":                lctx.botRecall,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// botRemember allows Lua scripts to remember a key-value pair with an optional shared flag.
func (lctx *luaContext) botRemember(L *glua.LState) int {
	r := lctx.getRobot(L, "Remember")
	key := L.CheckString(2)
	value := L.CheckString(3)
	shared := lctx.GetDefaultBool(4, false)

	if key == "" {
		L.RaiseError("Remember: key must not be empty")
		return 0
	}

	if value == "" {
		L.RaiseError("Remember: value must not be empty")
		return 0
	}

	r.Remember(key, value, shared)
	return 0
}

// botRememberThread remembers a key-value pair in a threaded context with an optional shared flag.
func (lctx *luaContext) botRememberThread(L *glua.LState) int {
	r := lctx.getRobot(L, "RememberThread")
	key := L.CheckString(2)
	value := L.CheckString(3)
	shared := lctx.GetDefaultBool(4, false)

	if key == "" {
		L.RaiseError("RememberThread: key must not be empty")
		return 0
	}

	if value == "" {
		L.RaiseError("RememberThread: value must not be empty")
		return 0
	}

	r.RememberThread(key, value, shared)
	return 0
}

// botRememberContext remembers a value within a specific context.
func (lctx *luaContext) botRememberContext(L *glua.LState) int {
	r := lctx.getRobot(L, "RememberContext")
	context := L.CheckString(2)
	value := L.CheckString(3)

	if context == "" {
		L.RaiseError("RememberContext: context must not be empty")
		return 0
	}

	if value == "" {
		L.RaiseError("RememberContext: value must not be empty")
		return 0
	}

	r.RememberContext(context, value)
	return 0
}

// botRememberContextThread remembers a value within a specific context in a threaded environment.
func (lctx *luaContext) botRememberContextThread(L *glua.LState) int {
	r := lctx.getRobot(L, "RememberContextThread")
	context := L.CheckString(2)
	value := L.CheckString(3)

	if context == "" {
		L.RaiseError("RememberContextThread: context must not be empty")
		return 0
	}

	if value == "" {
		L.RaiseError("RememberContextThread: value must not be empty")
		return 0
	}

	r.RememberContextThread(context, value)
	return 0
}

// botRecall recalls a value by key with an optional shared flag.
func (lctx *luaContext) botRecall(L *glua.LState) int {
	r := lctx.getRobot(L, "Recall")
	key := L.CheckString(2)
	shared := lctx.GetDefaultBool(3, false)

	if key == "" {
		L.RaiseError("Recall: key must not be empty")
		return 0
	}

	value := r.Recall(key, shared)
	L.Push(glua.LString(value))
	return 1
}

// GetDefaultBool retrieves a boolean argument from the Lua stack with a default value.
// If the argument at the given index is not a boolean, it returns the provided default.
func (lctx *luaContext) GetDefaultBool(index int, defaultVal bool) bool {
	if lctx.L.Get(index).Type() == glua.LTBool {
		return bool(lctx.L.CheckBool(index))
	}
	return defaultVal
}
