package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// botGetTaskConfig(luaState) -> (luaTableOrNil, retVal)
//
// Usage in Lua:
//
//	local cfg, retVal = bot:GetTaskConfig()
//	if retVal == ret.Ok then
//	  -- cfg is a Lua table (array or map) containing the plugin/job config
//	end
func (lctx luaContext) botGetTaskConfig(L *glua.LState) int {
	ud := L.CheckUserData(1)
	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("botGetTaskConfig")
		return pushFail(L)
	}

	// 1) First, try to unmarshal into map[string]interface{}
	mapConfig := make(map[string]interface{})
	retVal := lr.r.GetTaskConfig(&mapConfig)
	if retVal == robot.Ok {
		// success -> parse to Lua
		luaVal, err := parseGoValueToLua(L, mapConfig)
		if err != nil {
			// This is unusual (bad JSON?), log and return DataFormatError
			lr.r.Log(robot.Error, fmt.Sprintf("Error converting map config to Lua: %v", err))
			L.Push(glua.LNil) // no config
			L.Push(glua.LNumber(robot.DataFormatError))
			return 2
		}
		L.Push(luaVal)
		L.Push(glua.LNumber(robot.Ok))
		return 2
	}

	// 2) If we get a ConfigUnmarshalError, try a []interface{} fallback
	if retVal == robot.ConfigUnmarshalError {
		var sliceConfig []interface{}
		retVal = lr.r.GetTaskConfig(&sliceConfig)
		if retVal == robot.Ok {
			luaVal, err := parseGoValueToLua(L, sliceConfig)
			if err != nil {
				lr.r.Log(robot.Error, fmt.Sprintf("Error converting slice config to Lua: %v", err))
				L.Push(glua.LNil)
				L.Push(glua.LNumber(robot.DataFormatError))
				return 2
			}
			L.Push(luaVal)
			L.Push(glua.LNumber(robot.Ok))
			return 2
		}
		// else fall through to final error
	}

	// 3) If still not Ok, or some other error code, just return nil + that code
	L.Push(glua.LNil)
	L.Push(glua.LNumber(retVal))
	return 2
}

// RegisterConfigMethods adds the bot.GetTaskConfig -> botGetTaskConfig binding
func (lctx luaContext) RegisterConfigMethod(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"GetTaskConfig": lctx.botGetTaskConfig,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}
