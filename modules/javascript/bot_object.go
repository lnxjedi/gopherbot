// bot_userdata.go
package javascript

import (
	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

// registerBotObject - exposes a global "bot" in JS with all the properties
// we need to implement a proper bot in JavaScript, and all the methods needed
// for the Gopherbot robot API.
func (ctx *jsContext) registerBotObject() {
	botObj := ctx.vm.NewObject()

	// Set string fields directly from ctx.bot
	keys := []string{"user", "user_id", "channel", "channel_id", "thread_id", "message_id", "protocol", "brain"}
	for _, key := range keys {
		if value, ok := ctx.bot[key]; ok {
			botObj.Set(key, value)
		}
	}

	// Set threaded_message boolean based on ctx.bot["GOPHER_THREADED_MESSAGE"]
	if ctx.bot["threaded_message"] == "true" {
		botObj.Set("threaded_message", true)
	} else {
		botObj.Set("threaded_message", false)
	}

	// Calling methods on a nil would cause a panic in Go, so we just stop with adding
	// empty properties - mainly we want the 'require' to succeed.
	if ctx.r == nil {
		ctx.vm.Set("BOT", botObj)
		return
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
	// Expose "BOT" as a global variable
	ctx.vm.Set("BOT", botObj)
}
