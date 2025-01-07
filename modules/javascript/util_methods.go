// util_methods.go
package javascript

import (
	"fmt"
	"reflect"

	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

// botRandomInt wraps r.RandomInt and returns a random integer up to n.
// Usage in JS:
//
//	let rand = bot.RandomInt(n);
func (jr *jsBot) botRandomInt(call goja.FunctionCall) goja.Value {
	const methodName = "RandomInt"

	// Validate and retrieve the 'n' argument
	n := jr.requireFloatArg(methodName, call, 0)
	nInt := int(n)

	// Call the Go method
	val := jr.r.RandomInt(nInt)

	// Return the random integer as a JS number
	return jr.ctx.vm.ToValue(val)
}

// botRandomString implements r.RandomString(...) and returns a random string from the provided array.
// Usage in JS:
//
//	let randStr = bot.RandomString(["apple", "banana", "cherry"]);
func (jr *jsBot) botRandomString(call goja.FunctionCall) goja.Value {
	const methodName = "RandomString"

	// Validate and retrieve the 'array' argument
	if len(call.Arguments) < 1 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s: missing 'array' argument", methodName)))
	}

	arrayArg := call.Arguments[0]

	// Check if the argument is an array by inspecting its exported type
	exportedType := arrayArg.ExportType()
	if exportedType.Kind() != reflect.Slice {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s: argument must be an array, got %s", methodName, exportedType.Kind().String())))
	}

	// Type assert the exported value to []interface{}
	exportedSlice, ok := arrayArg.Export().([]interface{})
	if !ok {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s: failed to convert argument to []interface{}", methodName)))
	}

	// Iterate over the slice and collect string elements
	var goSlice []string
	for i, elem := range exportedSlice {
		str, ok := elem.(string)
		if ok {
			goSlice = append(goSlice, str)
		} else {
			jr.log(robot.Error, fmt.Sprintf("RandomString: non-string element at index %d, ignoring", i))
		}
	}

	if len(goSlice) == 0 {
		jr.log(robot.Error, "RandomString found no valid strings, returning empty string")
		return jr.ctx.vm.ToValue("")
	}

	// Call the Go method
	str := jr.r.RandomString(goSlice)

	// Optionally, log the selected random string
	jr.log(robot.Debug, fmt.Sprintf("RandomString: Selected '%s' from array", str))

	// Return the random string as a JS string
	return jr.ctx.vm.ToValue(str)
}

// botPause wraps r.Pause(...) and pauses execution for the specified number of seconds.
// Usage in JS:
//
//	let retVal = bot.Pause(5);
func (jr *jsBot) botPause(call goja.FunctionCall) goja.Value {
	const methodName = "Pause"

	// Validate and retrieve the 'seconds' argument
	sec := jr.requireFloatArg(methodName, call, 0)

	// Call the Go method
	jr.r.Pause(sec)

	// Return the retVal as a JS number
	return goja.Null()
}

// botCheckAdmin checks if the current user has administrative privileges.
// Usage in JS:
//
//	let isAdmin = bot.CheckAdmin();
func (jr *jsBot) botCheckAdmin(call goja.FunctionCall) goja.Value {
	const methodName = "CheckAdmin"

	// No arguments expected
	if len(call.Arguments) != 0 {
		jr.log(robot.Error, fmt.Sprintf("%s: expected 0 arguments, got %d", methodName, len(call.Arguments)))
		return jr.ctx.vm.ToValue(false)
	}

	// Call the Go method
	isAdmin := jr.r.CheckAdmin()

	// Return the boolean value
	return jr.ctx.vm.ToValue(isAdmin)
}

// botElevate elevates the current user's privileges, optionally forcing a 2FA prompt.
// Usage in JS:
//
//	let success = bot.Elevate(true);
func (jr *jsBot) botElevate(call goja.FunctionCall) goja.Value {
	const methodName = "Elevate"

	// Validate and retrieve the 'immediate' argument (optional)
	immediate := false
	if len(call.Arguments) > 0 {
		arg := call.Arguments[0]
		switch v := arg.Export().(type) {
		case bool:
			immediate = v
		default:
			jr.log(robot.Error, fmt.Sprintf("%s: 'immediate' argument must be a boolean, defaulting to false", methodName))
		}
	}

	// Call the Go method
	success := jr.r.Elevate(immediate)

	// Return the boolean value
	return jr.ctx.vm.ToValue(success)
}

// botLog logs a message at the specified log level.
// Usage in JS:
//
//	let retVal = bot.Log(log.Debug, "This is a debug message");
func (jr *jsBot) botLog(call goja.FunctionCall) goja.Value {
	const methodName = "Log"

	// Validate and retrieve the 'level' argument
	if len(call.Arguments) < 2 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s: requires 2 arguments (level, message)", methodName)))
	}

	levelArg := call.Arguments[0]
	level, ok := levelArg.Export().(int64)
	if !ok {
		jr.log(robot.Error, fmt.Sprintf("%s: 'level' argument must be a number, got: %s", methodName, levelArg.ExportType().String()))
		return jr.ctx.vm.ToValue(robot.Fail)
	}

	// Validate and retrieve the 'message' argument
	msgArg := call.Arguments[1]
	msg, ok := msgArg.Export().(string)
	if !ok {
		jr.log(robot.Error, fmt.Sprintf("%s: 'message' argument must be a string", methodName))
		return jr.ctx.vm.ToValue(robot.Fail)
	}

	// Call the Go method
	jr.r.Log(robot.LogLevel(level), msg)

	// Assuming Log doesn't fail, return robot.Ok
	return jr.ctx.vm.ToValue(robot.Ok)
}
