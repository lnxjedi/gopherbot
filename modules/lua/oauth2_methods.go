package lua

import (
	"encoding/json"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

func (lctx *luaContext) RegisterOAuth2Methods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"GetIdentityCredential": lctx.botGetIdentityCredential,
		"LinkOAuth2Identity":    lctx.botLinkOAuth2Identity,
		"UnlinkIdentity":        lctx.botUnlinkIdentity,
	}
	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

func (lctx *luaContext) botGetIdentityCredential(L *glua.LState) int {
	r := lctx.getRobot(L, "GetIdentityCredential")
	provider := L.CheckString(2)
	user := L.CheckString(3)
	credential, ret := r.GetIdentityCredential(provider, user)
	luaValue, err := parseGoValueToLua(L, credential)
	if err != nil {
		L.Push(glua.LNil)
		L.Push(glua.LNumber(robot.Failed))
		return 2
	}
	L.Push(luaValue)
	L.Push(glua.LNumber(ret))
	return 2
}

func (lctx *luaContext) botLinkOAuth2Identity(L *glua.LState) int {
	r := lctx.getRobot(L, "LinkOAuth2Identity")
	linkTable := L.CheckTable(2)
	goDatum, err := parseLuaValueToGo(linkTable, map[*glua.LTable]bool{})
	if err != nil {
		L.Push(glua.LNumber(robot.IdentityInvalidLinkRequest))
		return 1
	}
	blob, err := json.Marshal(goDatum)
	if err != nil {
		L.Push(glua.LNumber(robot.IdentityInvalidLinkRequest))
		return 1
	}
	var req robot.OAuth2IdentityLinkRequest
	if err := json.Unmarshal(blob, &req); err != nil {
		L.Push(glua.LNumber(robot.IdentityInvalidLinkRequest))
		return 1
	}
	L.Push(glua.LNumber(r.LinkOAuth2Identity(&req)))
	return 1
}

func (lctx *luaContext) botUnlinkIdentity(L *glua.LState) int {
	r := lctx.getRobot(L, "UnlinkIdentity")
	provider := L.CheckString(2)
	user := L.CheckString(3)
	L.Push(glua.LNumber(r.UnlinkIdentity(provider, user)))
	return 1
}
