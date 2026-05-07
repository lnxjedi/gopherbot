package lua

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cjoudrey/gluahttp"
	glua "github.com/yuin/gopher-lua"
)

func registerHttpModule(L *glua.LState) {
	L.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	L.PreloadModule("json", jsonModuleLoader)
}

func jsonModuleLoader(L *glua.LState) int {
	mod := L.SetFuncs(L.NewTable(), map[string]glua.LGFunction{
		"decode": luaJSONDecode,
		"encode": luaJSONEncode,
	})
	L.Push(mod)
	return 1
}

func luaJSONDecode(L *glua.LState) int {
	var decoded interface{}
	if err := json.Unmarshal([]byte(L.CheckString(1)), &decoded); err != nil {
		return pushLuaStringError(L, fmt.Errorf("json.decode: %w", err))
	}
	luaValue, err := parseGoValueToLua(L, decoded)
	if err != nil {
		return pushLuaStringError(L, err)
	}
	L.Push(luaValue)
	return 1
}

func luaJSONEncode(L *glua.LState) int {
	converted, err := parseLuaValueToGo(L.CheckAny(1), map[*glua.LTable]bool{})
	if err != nil {
		return pushLuaStringError(L, err)
	}
	data, err := json.Marshal(converted)
	if err != nil {
		return pushLuaStringError(L, err)
	}
	L.Push(glua.LString(data))
	return 1
}

func pushLuaStringError(L *glua.LState, err error) int {
	L.Push(glua.LNil)
	L.Push(glua.LString(err.Error()))
	return 2
}
