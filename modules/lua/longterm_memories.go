package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterLongTermMemoryMethods adds CheckoutDatum, UpdateDatum, and CheckinDatum
// to the bot's metatable.
func (lctx luaContext) RegisterLongTermMemoryMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"CheckoutDatum": lctx.botCheckoutDatum,
		"UpdateDatum":   lctx.botUpdateDatum,
		"CheckinDatum":  lctx.botCheckinDatum,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// -------------------------------------------------------------------
// 1. botCheckoutDatum
// -------------------------------------------------------------------

// botCheckoutDatum allows Lua scripts to checkout a datum by key, optionally read/write.
func (lctx luaContext) botCheckoutDatum(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.CheckString(2)
	rwLua := L.Get(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("CheckoutDatum")
		return pushFail(L)
	}

	rw := false
	if b, isBool := rwLua.(glua.LBool); isBool {
		rw = bool(b)
	}

	var datum interface{}
	lockToken, exists, ret := lr.r.CheckoutDatum(key, &datum, rw)
	if ret != robot.Ok {
		L.Push(glua.LNumber(ret))
		L.Push(glua.LNil)
		L.Push(glua.LString(""))
		return 3
	}
	if !exists {
		// Return empty table if the key doesn't exist
		L.Push(glua.LNumber(robot.Ok))
		L.Push(L.CreateTable(0, 0))
		L.Push(glua.LString(lockToken))
		return 3
	}

	luaValue, err := parseGoValueToLua(L, datum)
	if err != nil {
		lr.r.Log(robot.Error, fmt.Sprintf("Lua error in CheckoutDatum '%s': %v", key, err))
		L.Push(glua.LNumber(robot.DataFormatError))
		L.Push(glua.LNil)
		L.Push(glua.LString(lockToken))
		return 3
	}

	L.Push(glua.LNumber(robot.Ok))
	L.Push(luaValue)
	L.Push(glua.LString(lockToken))
	return 3
}

// -------------------------------------------------------------------
// 2. botUpdateDatum, botCheckinDatum
// -------------------------------------------------------------------

// botUpdateDatum allows Lua scripts to update a datum by key and lockToken.
func (lctx luaContext) botUpdateDatum(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.CheckString(2)
	lockToken := L.CheckString(3)
	luaDataVal := L.Get(4)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("UpdateDatum")
		return pushFail(L)
	}

	visited := make(map[*glua.LTable]bool)
	goDatum, err := parseLuaValueToGo(luaDataVal, visited)
	if err != nil {
		lr.r.Log(robot.Error, fmt.Sprintf("Error serializing Lua object for key '%s': %v", key, err))
		L.Push(glua.LNumber(robot.DataFormatError))
		return 1
	}

	retVal := lr.r.UpdateDatum(key, lockToken, goDatum)
	L.Push(glua.LNumber(retVal))
	return 1
}

// botCheckinDatum allows Lua scripts to checkin a datum by key and lockToken.
func (lctx luaContext) botCheckinDatum(L *glua.LState) int {
	ud := L.CheckUserData(1)
	key := L.CheckString(2)
	lockToken := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lctx.logBotErr("CheckinDatum")
		return pushFail(L)
	}

	lr.r.CheckinDatum(key, lockToken)
	L.Push(glua.LNumber(robot.Ok))
	return 1
}
