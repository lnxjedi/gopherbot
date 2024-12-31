package javascript

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

// botFixed(bot:Fixed())
//
// Creates a new Robot that uses "fixed" message format, returning a
// fresh JS bot object. For example:
//
//	let fbot = bot.Fixed()
//	fbot.Say("This uses fixed-width formatting if the connector supports it")
func (jr *jsBot) botFixed(call goja.FunctionCall) goja.Value {
	fixedBot := jr.r.Fixed()
	newBot := &jsBot{
		r:   fixedBot,
		ctx: jr.ctx,
	}
	return newBot.createBotObject()
}

// botDirect(bot:Direct())
//
// Creates a new Robot set up to DM the user (no channel). Example:
//
//	let dbot = bot.Direct()
//	dbot.Say("This sends a direct message to the user!")
func (jr *jsBot) botDirect(call goja.FunctionCall) goja.Value {
	directBot := jr.r.Direct()
	newBot := &jsBot{
		r:   directBot,
		ctx: jr.ctx,
	}
	obj := newBot.createBotObject()
	// As in your Lua code, we can optionally blank out the channel fields.
	// The JavaScript object can store them as empty strings to reflect
	// that it's a DM context now.
	obj.Set("channel", "")
	obj.Set("channel_id", "")
	return obj
}

// botThreaded(bot:Threaded())
//
// Creates a new Robot for replying in a thread. Example:
//
//	let tbot = bot.Threaded()
//	tbot.Say("This will reply in a thread if the connector supports it")
func (jr *jsBot) botThreaded(call goja.FunctionCall) goja.Value {
	threadedBot := jr.r.Threaded()
	newBot := &jsBot{
		r:   threadedBot,
		ctx: jr.ctx,
	}
	obj := newBot.createBotObject()
	// As in Lua, set a "threaded" field to true on the new object
	obj.Set("threaded", true)
	return obj
}

// botMessageFormat(bot:MessageFormat(fmt.Variable))
//
// Returns a new Robot with the specified message format. For instance:
//
//	let vbot = bot.MessageFormat(fmt.Variable)
//	vbot.Say("Now in variable-width format, if supported")
func (jr *jsBot) botMessageFormat(call goja.FunctionCall) goja.Value {
	const methodName = "MessageFormat"

	// We need a numeric argument. If you already wrote a 'requireNumberArg' helper,
	// use that. Otherwise we can do a simple check inline:
	if len(call.Arguments) < 1 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s requires a numeric argument", methodName)))
	}

	rawVal := call.Arguments[0].Export()
	// Accept int64, float64, etc., then cast to int:
	var formatInt int
	switch n := rawVal.(type) {
	case int64:
		formatInt = int(n)
	case float64:
		formatInt = int(n)
	default:
		panic(jr.ctx.vm.ToValue(fmt.Sprintf(
			"%s: argument must be a number (use fmt.Raw, fmt.Fixed, or fmt.Variable)",
			methodName,
		)))
	}

	// Optional: verify that formatInt is a valid value in [0..2]
	// (Raw=0, Fixed=1, Variable=2)
	if formatInt < 0 || formatInt > 2 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf(
			"Invalid message format value: %d. Must be Raw=0, Fixed=1, or Variable=2",
			formatInt,
		)))
	}

	// Use the robot's MessageFormat method to create the updated bot
	formatted := jr.r.MessageFormat(robot.MessageFormat(formatInt))
	newBot := &jsBot{
		r:   formatted,
		ctx: jr.ctx,
	}
	return newBot.createBotObject()
}
