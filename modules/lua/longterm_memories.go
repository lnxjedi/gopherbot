package lua

import (
	"encoding/json"
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

// -------------------------------------------------------------------
// 3. Converting Lua -> Go (cycle detection + array vs. map logic)
// -------------------------------------------------------------------

func parseLuaValueToGo(val glua.LValue, visited map[*glua.LTable]bool) (interface{}, error) {
	switch converted := val.(type) {
	case *glua.LNilType:
		return nil, nil
	case glua.LBool:
		return bool(converted), nil
	case glua.LNumber:
		// Distinguish integer from float
		n := float64(converted)
		i := int64(n)
		if float64(i) == n {
			return i, nil
		}
		return n, nil
	case glua.LString:
		return string(converted), nil
	case *glua.LTable:
		if visited[converted] {
			return nil, fmt.Errorf("cycle detected in Lua table")
		}
		visited[converted] = true
		return parseLuaTableToGo(converted, visited)
	default:
		return nil, fmt.Errorf("unsupported Lua type %T", val)
	}
}

func parseLuaTableToGo(tbl *glua.LTable, visited map[*glua.LTable]bool) (interface{}, error) {
	mappedPairs := make(map[glua.LValue]glua.LValue)
	tbl.ForEach(func(k, v glua.LValue) {
		mappedPairs[k] = v
	})

	// Check if it's an array
	arrayCandidate := true
	keys := []int{}
	maxKey := -1

	for k := range mappedPairs {
		switch knum := k.(type) {
		case glua.LNumber:
			ik := int(knum)
			if float64(ik) != float64(knum) {
				arrayCandidate = false
				continue
			}
			if ik > maxKey {
				maxKey = ik
			}
			keys = append(keys, ik)
		default:
			arrayCandidate = false
		}
	}

	if arrayCandidate {
		if len(keys) == 0 {
			return []interface{}{}, nil
		}

		minKey := maxKey
		for _, k := range keys {
			if k < minKey {
				minKey = k
			}
		}
		// Must start at 0 or 1, no holes
		if minKey != 0 && minKey != 1 {
			return nil, fmt.Errorf("array must start at 0 or 1, got %d", minKey)
		}
		expectedCount := (maxKey - minKey) + 1
		if len(keys) != expectedCount {
			return nil, fmt.Errorf("sparse array not handled: holes in indexes")
		}

		arr := make([]interface{}, expectedCount)
		for i := minKey; i <= maxKey; i++ {
			vVal, ok := mappedPairs[glua.LNumber(i)]
			if !ok {
				return nil, fmt.Errorf("sparse array not handled: missing index %d", i)
			}
			goVal, err := parseLuaValueToGo(vVal, visited)
			if err != nil {
				return nil, err
			}
			arr[i-minKey] = goVal
		}
		return arr, nil
	}

	// Otherwise, treat as map
	result := make(map[string]interface{}, len(mappedPairs))
	for kVal, vVal := range mappedPairs {
		goKey := kVal.String() // safe, because kVal is an LValue
		goVal, err := parseLuaValueToGo(vVal, visited)
		if err != nil {
			return nil, err
		}
		result[goKey] = goVal
	}
	return result, nil
}

// -------------------------------------------------------------------
// 4. Converting Go -> Lua
// -------------------------------------------------------------------

func parseGoValueToLua(L *glua.LState, data interface{}) (glua.LValue, error) {
	switch val := data.(type) {
	case nil:
		return glua.LNil, nil
	case bool:
		return glua.LBool(val), nil
	case float64:
		return glua.LNumber(val), nil
	case float32:
		return glua.LNumber(val), nil
	case int:
		return glua.LNumber(float64(val)), nil
	case int32:
		return glua.LNumber(float64(val)), nil
	case int64:
		return glua.LNumber(float64(val)), nil
	case uint:
		return glua.LNumber(float64(val)), nil
	case uint32:
		return glua.LNumber(float64(val)), nil
	case uint64:
		return glua.LNumber(float64(val)), nil
	case string:
		return glua.LString(val), nil
	case []interface{}:
		tbl := L.CreateTable(len(val), 0)
		for i, elem := range val {
			lv, err := parseGoValueToLua(L, elem)
			if err != nil {
				return glua.LNil, err
			}
			tbl.RawSetInt(i+1, lv)
		}
		return tbl, nil
	case map[string]interface{}:
		tbl := L.CreateTable(0, len(val))
		for k, elem := range val {
			lv, err := parseGoValueToLua(L, elem)
			if err != nil {
				return glua.LNil, err
			}
			tbl.RawSetString(k, lv)
		}
		return tbl, nil
	default:
		// If it's some other numeric type or struct, do a JSON round-trip
		b, err := json.Marshal(val)
		if err != nil {
			return glua.LNil, err
		}
		var tmp interface{}
		if err := json.Unmarshal(b, &tmp); err != nil {
			return glua.LNil, err
		}
		return parseGoValueToLua(L, tmp)
	}
}
