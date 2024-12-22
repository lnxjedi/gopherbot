package lua

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterUtilMethods adds RandomInt, RandomString, and Pause to the robot's metatable
func RegisterUtilMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"RandomInt":    robotRandomInt,
		"RandomString": robotRandomString,
		"Pause":        robotPause,
	}
	robotIndex := getRobotMethodTable(L)
	L.SetFuncs(robotIndex, methods)
}

// robotRandomInt(luaState) -> int
//
// Usage in Lua:
//
//	local x = robot:RandomInt(10)  -- x in [0..9]
//
// If invalid input is provided, logs an error and returns 0.
func robotRandomInt(L *glua.LState) int {
	// Arg 1: robot userdata, Arg 2: upperBound
	ud := L.CheckUserData(1)
	nLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lr.r.Log(robot.Error, "RandomInt called with invalid robot userdata")
		L.Push(glua.LNumber(0))
		return 1
	}

	if nLua.Type() != glua.LTNumber {
		lr.r.Log(robot.Error, fmt.Sprintf("RandomInt requires a numeric argument, got %s", nLua.Type().String()))
		L.Push(glua.LNumber(0))
		return 1
	}

	n := int(nLua.(glua.LNumber))
	val := lr.r.RandomInt(n)

	L.Push(glua.LNumber(val))
	return 1
}

// robotRandomString(luaState) -> string
//
// Usage in Lua:
//
//	local arr = { "hello", "there", "people" }
//	local s = robot:RandomString(arr)
//	-- s is one of the above
//
// If invalid input or array is empty, logs an error and returns "".
func robotRandomString(L *glua.LState) int {
	// Arg 1: robot userdata, Arg 2: table of strings
	ud := L.CheckUserData(1)
	arrLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lr.r.Log(robot.Error, "RandomString called with invalid robot userdata")
		L.Push(glua.LString(""))
		return 1
	}

	// Must be a table
	if arrLua.Type() != glua.LTTable {
		lr.r.Log(robot.Error, "RandomString called with non-table argument")
		L.Push(glua.LString(""))
		return 1
	}

	tbl := arrLua.(*glua.LTable)
	length := tbl.Len() // 1-based length

	var goSlice []string
	for i := 1; i <= length; i++ {
		val := tbl.RawGetInt(i)
		if val.Type() == glua.LTString {
			goSlice = append(goSlice, val.String())
		} else {
			lr.r.Log(robot.Error, fmt.Sprintf(
				"RandomString: non-string element at index %d, ignoring", i))
		}
	}

	if len(goSlice) == 0 {
		lr.r.Log(robot.Error, "RandomString found no valid strings, returning empty string")
		L.Push(glua.LString(""))
		return 1
	}

	str := lr.r.RandomString(goSlice)
	L.Push(glua.LString(str))
	return 1
}

// robotPause(luaState) -> (no return value)
//
// Usage in Lua:
//
//	robot:Pause(2.5)  -- sleeps 2.5 seconds
//
// If invalid input is provided, logs an error and still returns no value.
func robotPause(L *glua.LState) int {
	ud := L.CheckUserData(1)
	secLua := L.Get(2)

	lr, ok := ud.Value.(*luaRobot)
	if !ok {
		lr.r.Log(robot.Error, "Pause called with invalid robot userdata")
		return 0
	}

	if secLua.Type() != glua.LTNumber {
		lr.r.Log(robot.Error, fmt.Sprintf("Pause requires a numeric argument, got %s", secLua.Type().String()))
		return 0
	}

	sec := float64(secLua.(glua.LNumber))
	// Use the robot's Pause method to handle fractional seconds
	lr.r.Pause(sec)

	// No return value
	return 0
}
