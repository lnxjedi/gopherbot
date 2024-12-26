// bot_userdata.go
package javascript

import (
	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

// registerRobotObject exposes a global "robot" in JS with a .New() method
// that returns a "bot" object. The "bot" object has direct access to fields
// and methods for interacting with Gopherbot (Say, Log, Memory, etc.).
func (ctx *jsContext) registerRobotObject() {
	obj := ctx.vm.NewObject()

	// Provide a "New()" function that returns a new "bot" object
	obj.Set("New", func(call goja.FunctionCall) goja.Value {
		botVal := ctx.newBotValue()
		return botVal
	})

	// Expose "robot" as a global variable
	ctx.vm.Set("robot", obj)
}

// newBotValue returns a goja.Value (object) with the fields and methods
// that replicate the "bot" userdatas from the Lua integration.
func (ctx *jsContext) newBotValue() goja.Value {
	botObj := ctx.vm.NewObject()

	// Set string fields directly from ctx.env
	botObj.Set("user", ctx.env["GOPHER_USER"])
	botObj.Set("user_id", ctx.env["GOPHER_USER_ID"])
	botObj.Set("channel", ctx.env["GOPHER_CHANNEL"])
	botObj.Set("channel_id", ctx.env["GOPHER_CHANNEL_ID"])
	botObj.Set("thread_id", ctx.env["GOPHER_THREAD_ID"])
	botObj.Set("message_id", ctx.env["GOPHER_MESSAGE_ID"])
	botObj.Set("protocol", ctx.env["GOPHER_PROTOCOL"])
	botObj.Set("brain", ctx.env["GOPHER_BRAIN"])

	// Set threaded_message boolean based on ctx.env["GOPHER_THREADED_MESSAGE"]
	if ctx.env["GOPHER_THREADED_MESSAGE"] == "true" {
		botObj.Set("threaded_message", true)
	} else {
		botObj.Set("threaded_message", false)
	}

	// Attach "Say" method, capturing the return value
	botObj.Set("Say", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return ctx.vm.ToValue(int(robot.Failed)) // Return status Invalid if no args
		}
		msg := call.Arguments[0].String()
		retVal := ctx.r.Say(msg)
		return ctx.vm.ToValue(int(retVal)) // Convert robot.ReturnCode to int
	})

	// Attach "Reply" method
	botObj.Set("Reply", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return ctx.vm.ToValue(int(robot.Failed)) // Return status Invalid if no args
		}
		msg := call.Arguments[0].String()

		// Get the "user" property from the bot object
		userVal := botObj.Get("user")
		if userVal == nil || userVal.ExportType().String() != "string" {
			return ctx.vm.ToValue(int(robot.Failed)) // Return status Failed if user is not a string
		}
		user := userVal.String()

		retVal := ctx.r.Reply(user, msg)
		return ctx.vm.ToValue(int(retVal)) // Convert robot.ReturnCode to int
	})

	// ... other methods (Log, Memory, etc.) ...

	return botObj
}
