// attribute_methods.go
package javascript

import (
	"fmt"

	"github.com/dop251/goja"
)

// botGetBotAttribute retrieves a bot attribute.
// Usage in JS:
//
//	let result = bot.GetBotAttribute("name");
//	console.log(result.attribute, result.retVal);
func (jr *jsBot) botGetBotAttribute(call goja.FunctionCall) goja.Value {
	const methodName = "GetBotAttribute"

	// Validate and retrieve the 'attribute' argument
	attribute := jr.requireStringArg(methodName, call, 0)

	// Check if attribute is empty
	if attribute == "" {
		panic(jr.ctx.vm.ToValue("GetBotAttribute: attribute must not be empty"))
	}

	// Call the Go method
	ret := jr.r.GetBotAttribute(attribute)

	// Create a JS object to return
	resultObj := jr.ctx.vm.NewObject()
	err := resultObj.Set("attribute", ret.Attribute)
	if err != nil {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("GetBotAttribute: failed to set 'attribute': %v", err)))
	}
	err = resultObj.Set("retVal", ret.RetVal)
	if err != nil {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("GetBotAttribute: failed to set 'retVal': %v", err)))
	}

	return resultObj
}

// botGetUserAttribute retrieves a user attribute.
// Usage in JS:
//
//	let result = bot.GetUserAttribute("user123", "role");
//	console.log(result.attribute, result.retVal);
func (jr *jsBot) botGetUserAttribute(call goja.FunctionCall) goja.Value {
	const methodName = "GetUserAttribute"

	// Validate and retrieve the 'user' argument
	user := jr.requireStringArg(methodName, call, 0)

	// Validate and retrieve the 'attribute' argument
	attribute := jr.requireStringArg(methodName, call, 1)

	// Check if user is empty
	if user == "" {
		panic(jr.ctx.vm.ToValue("GetUserAttribute: user must not be empty"))
	}
	// Check if attribute is empty
	if attribute == "" {
		panic(jr.ctx.vm.ToValue("GetUserAttribute: attribute must not be empty"))
	}

	// Call the Go method
	ret := jr.r.GetUserAttribute(user, attribute)

	// Create a JS object to return
	resultObj := jr.ctx.vm.NewObject()
	err := resultObj.Set("attribute", ret.Attribute)
	if err != nil {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("GetUserAttribute: failed to set 'attribute': %v", err)))
	}
	err = resultObj.Set("retVal", ret.RetVal)
	if err != nil {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("GetUserAttribute: failed to set 'retVal': %v", err)))
	}

	return resultObj
}

// botGetSenderAttribute retrieves an attribute of the message sender.
// Usage in JS:
//
//	let result = bot.GetSenderAttribute("status");
//	console.log(result.attribute, result.retVal);
func (jr *jsBot) botGetSenderAttribute(call goja.FunctionCall) goja.Value {
	const methodName = "GetSenderAttribute"

	// Validate and retrieve the 'attribute' argument
	attribute := jr.requireStringArg(methodName, call, 0)

	// Check if attribute is empty
	if attribute == "" {
		panic(jr.ctx.vm.ToValue("GetSenderAttribute: attribute must not be empty"))
	}

	// Call the Go method
	ret := jr.r.GetSenderAttribute(attribute)

	// Create a JS object to return
	resultObj := jr.ctx.vm.NewObject()
	err := resultObj.Set("attribute", ret.Attribute)
	if err != nil {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("GetSenderAttribute: failed to set 'attribute': %v", err)))
	}
	err = resultObj.Set("retVal", ret.RetVal)
	if err != nil {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("GetSenderAttribute: failed to set 'retVal': %v", err)))
	}

	return resultObj
}
