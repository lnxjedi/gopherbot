package javascript

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

// botCheckoutDatum(bot:CheckoutDatum(key, rw?))
//
// JavaScript usage example:
//
//	let result = bot.CheckoutDatum("myKey", true); // read/write
//	if (result.retVal === ret.Ok && result.exists) {
//	    // result.datum is the existing data
//	} else {
//	    // handle not found or other retVal
//	}
func (jr *jsBot) botCheckoutDatum(call goja.FunctionCall) goja.Value {
	const methodName = "CheckoutDatum"

	// 1) Validate arguments
	key := jr.requireStringArg(methodName, call, 0)

	// Arg #1 (rw) is optional; if missing or not a bool, default false
	var rw bool
	if len(call.Arguments) > 1 {
		// Attempt to interpret the second argument as a JS boolean
		rawVal := call.Arguments[1].Export()
		if b, ok := rawVal.(bool); ok {
			rw = b
		} else if rawVal != nil {
			// If it’s not nil or bool, we panic
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: invalid value for 'rw' (must be boolean or omitted)",
				methodName,
			)))
		}
	}

	// 2) Perform CheckoutDatum on the Go side
	var goDatum interface{}
	lockToken, exists, retVal := jr.r.CheckoutDatum(key, &goDatum, rw)

	// If the robot returned an error code (e.g. user error, etc.), return an object
	// with retVal and empty fields
	if retVal != robot.Ok {
		resultObj := jr.ctx.vm.NewObject()
		resultObj.Set("retVal", int(retVal))
		resultObj.Set("exists", false)
		resultObj.Set("datum", goja.Null())
		resultObj.Set("token", "")
		return resultObj
	}

	// Convert the Go interface{} => a JS value. For simple use, just do .ToValue().
	// If there's cyclical data, that can cause an error or produce an object
	// that might not be fully serializable. For now, we assume no cycles.
	datumVal, err := parseGoValueToJS(jr.ctx.vm, goDatum)
	if err != nil {
		// If we can’t parse, return DataFormatError
		jr.ctx.l.Log(robot.Error, fmt.Sprintf("JavaScript error in CheckoutDatum for key '%s': %v", key, err))
		resultObj := jr.ctx.vm.NewObject()
		resultObj.Set("retVal", int(robot.DataFormatError))
		resultObj.Set("exists", false)
		resultObj.Set("datum", goja.Null())
		resultObj.Set("token", lockToken)
		return resultObj
	}

	// 3) Build a result object that the JS script can handle
	resultObj := jr.ctx.vm.NewObject()
	resultObj.Set("retVal", int(retVal))
	resultObj.Set("exists", exists)
	if exists {
		resultObj.Set("datum", datumVal)
	} else {
		// If the key doesn't exist in the database, return an empty object or undefined
		resultObj.Set("datum", jr.ctx.vm.NewObject())
	}
	resultObj.Set("token", lockToken)

	return resultObj
}

// botUpdateDatum(bot:UpdateDatum(memoryObj))
//
// JavaScript usage example:
//
//	let out = bot.CheckoutDatum("myKey", true);
//	let memory = out.datum;
//	memory.newField = 42;
//	let retVal = bot.UpdateDatum({ key: "myKey", token: out.token, datum: memory });
func (jr *jsBot) botUpdateDatum(call goja.FunctionCall) goja.Value {
	const methodName = "UpdateDatum"

	if len(call.Arguments) < 1 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s: requires a memory object", methodName)))
	}
	memObj := call.Arguments[0].ToObject(jr.ctx.vm)

	// Extract key, token, datum from the JS memory object
	keyVal := memObj.Get("key").Export()
	tokenVal := memObj.Get("token").Export()
	datumVal := memObj.Get("datum")

	keyStr, okKey := keyVal.(string)
	tokenStr, okTok := tokenVal.(string)
	if !okKey || !okTok || keyStr == "" || tokenStr == "" {
		panic(jr.ctx.vm.ToValue(
			"UpdateDatum requires a memory object with non-empty 'key' and 'token' fields",
		))
	}

	// Convert the JS 'datum' back into Go
	goDatum, err := parseJSValueToGo(datumVal)
	if err != nil {
		jr.ctx.l.Log(robot.Error, fmt.Sprintf("Error serializing JS object for key '%s': %v", keyStr, err))
		return jr.ctx.vm.ToValue(int(robot.DataFormatError))
	}

	// Call the underlying Go method
	retVal := jr.r.UpdateDatum(keyStr, tokenStr, goDatum)
	return jr.ctx.vm.ToValue(int(retVal))
}

// botCheckinDatum(bot:CheckinDatum(memoryObj))
//
// JavaScript usage example:
//
//	bot.CheckinDatum({ key: "myKey", token: out.token });
func (jr *jsBot) botCheckinDatum(call goja.FunctionCall) goja.Value {
	const methodName = "CheckinDatum"

	if len(call.Arguments) < 1 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s: requires a memory object", methodName)))
	}
	memObj := call.Arguments[0].ToObject(jr.ctx.vm)

	keyVal := memObj.Get("key").Export()
	tokenVal := memObj.Get("token").Export()

	keyStr, okKey := keyVal.(string)
	tokenStr, okTok := tokenVal.(string)
	if !okKey || !okTok || keyStr == "" || tokenStr == "" {
		panic(jr.ctx.vm.ToValue(
			"CheckinDatum requires a memory object with non-empty 'key' and 'token' fields",
		))
	}

	// Just call CheckinDatum
	jr.r.CheckinDatum(keyStr, tokenStr)
	// Return robot.Ok as an int
	return jr.ctx.vm.ToValue(int(robot.Ok))
}

// parseGoValueToJS converts a Go interface{} into a goja.Value. If you have cyclical
// data structures, you’d need extra checks. For now, we just do a naive approach.
func parseGoValueToJS(rt *goja.Runtime, data interface{}) (goja.Value, error) {
	// For many use cases, simply rt.ToValue(...) is enough.
	val := rt.ToValue(data)
	return val, nil
}

// parseJSValueToGo converts a goja.Value into an interface{} suitable for JSON
// serialization. If you expect complex objects or cyclical references from JS,
// you’d need specialized logic. Here we assume no cycles or functions to store.
func parseJSValueToGo(v goja.Value) (interface{}, error) {
	return v.Export(), nil
}
