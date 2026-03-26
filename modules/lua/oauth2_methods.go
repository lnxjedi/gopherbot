package lua

import (
	"encoding/json"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

func (lctx *luaContext) RegisterOAuth2Methods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"GetOAuth2Token":   lctx.botGetOAuth2Token,
		"LinkOAuth2User":   lctx.botLinkOAuth2User,
		"UnlinkOAuth2User": lctx.botUnlinkOAuth2User,
	}
	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

func (lctx *luaContext) botGetOAuth2Token(L *glua.LState) int {
	r := lctx.getRobot(L, "GetOAuth2Token")
	provider := L.CheckString(2)
	user := L.CheckString(3)
	token, ret := r.GetOAuth2Token(provider, user)
	L.Push(glua.LString(token))
	L.Push(glua.LNumber(ret))
	return 2
}

func (lctx *luaContext) botLinkOAuth2User(L *glua.LState) int {
	r := lctx.getRobot(L, "LinkOAuth2User")
	linkTable := L.CheckTable(2)
	goDatum, err := parseLuaValueToGo(linkTable, map[*glua.LTable]bool{})
	if err != nil {
		L.Push(glua.LNumber(robot.OAuth2InvalidLinkRequest))
		return 1
	}
	blob, err := json.Marshal(goDatum)
	if err != nil {
		L.Push(glua.LNumber(robot.OAuth2InvalidLinkRequest))
		return 1
	}
	var req robot.OAuth2LinkRequest
	if err := json.Unmarshal(blob, &req); err != nil {
		L.Push(glua.LNumber(robot.OAuth2InvalidLinkRequest))
		return 1
	}
	L.Push(glua.LNumber(r.LinkOAuth2User(&req)))
	return 1
}

func (lctx *luaContext) botUnlinkOAuth2User(L *glua.LState) int {
	r := lctx.getRobot(L, "UnlinkOAuth2User")
	provider := L.CheckString(2)
	user := L.CheckString(3)
	L.Push(glua.LNumber(r.UnlinkOAuth2User(provider, user)))
	return 1
}
