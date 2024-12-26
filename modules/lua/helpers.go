package lua

import (
	"encoding/json"
	"fmt"

	glua "github.com/yuin/gopher-lua"
)

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

// parseLuaValueToGo - Converting Lua -> Go (cycle detection + array vs. map logic)
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

// parseLuaTableToGo - Converting Lua -> Go (cycle detection + array vs. map logic)
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
// Converting Go -> Lua
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
