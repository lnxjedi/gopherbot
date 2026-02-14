// pipeline_methods.go
package javascript

import (
	"fmt"

	"github.com/dop251/goja"
)

// botGetParameter(bot:GetParameter(name) -> string)
func (jr *jsBot) botGetParameter(call goja.FunctionCall) goja.Value {
	const methodName = "GetParameter"

	// Argument #0: name (string)
	name := jr.requireStringArg(methodName, call, 0)

	if name == "" {
		panic(jr.ctx.vm.ToValue("GetParameter: name must not be empty"))
	}

	// Call the underlying Go method
	val := jr.r.GetParameter(name)

	// Return the value as a string
	return jr.ctx.vm.ToValue(val)
}

// botSetParameter(bot:SetParameter(name, value) -> bool)
func (jr *jsBot) botSetParameter(call goja.FunctionCall) goja.Value {
	const methodName = "SetParameter"

	// Argument #0: name (string)
	name := jr.requireStringArg(methodName, call, 0)

	// Argument #1: value (string)
	value := jr.requireStringArg(methodName, call, 1)

	if name == "" {
		panic(jr.ctx.vm.ToValue("SetParameter: name must not be empty"))
	}
	if value == "" {
		panic(jr.ctx.vm.ToValue("SetParameter: value must not be empty"))
	}

	// Call the underlying Go method
	okSet := jr.r.SetParameter(name, value)

	// Return the result as a boolean
	return jr.ctx.vm.ToValue(okSet)
}

// botSubscribe(bot:Subscribe() -> bool)
func (jr *jsBot) botSubscribe(call goja.FunctionCall) goja.Value {
	success := jr.r.Subscribe()
	return jr.ctx.vm.ToValue(success)
}

// botUnsubscribe(bot:Unsubscribe() -> bool)
func (jr *jsBot) botUnsubscribe(call goja.FunctionCall) goja.Value {
	success := jr.r.Unsubscribe()
	return jr.ctx.vm.ToValue(success)
}

// botExclusive(bot:Exclusive(tag, queueTask) -> bool)
func (jr *jsBot) botExclusive(call goja.FunctionCall) goja.Value {
	const methodName = "Exclusive"

	// Argument #0: tag (string)
	tag := jr.requireStringArg(methodName, call, 0)

	// Argument #1: queueTask (boolean)
	if len(call.Arguments) <= 1 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf(
			"%s: missing argument #2 (queueTask)",
			methodName,
		)))
	}
	queueTaskVal := call.Arguments[1].Export()
	queueTask, ok := queueTaskVal.(bool)
	if !ok {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf(
			"%s: argument #2 (queueTask) must be a boolean",
			methodName,
		)))
	}

	if tag == "" {
		panic(jr.ctx.vm.ToValue("Exclusive: tag must not be empty"))
	}

	// Call the underlying Go method
	success := jr.r.Exclusive(tag, queueTask)

	// Return the result as a boolean
	return jr.ctx.vm.ToValue(success)
}

// botSpawnJob(bot:SpawnJob(name, ...args) -> RetVal)
func (jr *jsBot) botSpawnJob(call goja.FunctionCall) goja.Value {
	const methodName = "SpawnJob"

	// Argument #0: name (string)
	name := jr.requireStringArg(methodName, call, 0)

	if name == "" {
		panic(jr.ctx.vm.ToValue("SpawnJob: name must not be empty"))
	}

	// Arguments #1..N: args (strings)
	extras := parseStringArgs(jr.ctx.vm, call, 1)

	// Call the underlying Go method
	ret := jr.r.SpawnJob(name, extras...)

	// Return the RetVal as an integer
	return jr.ctx.vm.ToValue(int(ret))
}

// botAddTask(bot:AddTask(name, ...args) -> RetVal)
func (jr *jsBot) botAddTask(call goja.FunctionCall) goja.Value {
	const methodName = "AddTask"

	// Argument #0: name (string)
	name := jr.requireStringArg(methodName, call, 0)

	if name == "" {
		panic(jr.ctx.vm.ToValue("AddTask: name must not be empty"))
	}

	// Arguments #1..N: args (strings)
	extras := parseStringArgs(jr.ctx.vm, call, 1)

	// Call the underlying Go method
	ret := jr.r.AddTask(name, extras...)

	// Return the RetVal as an integer
	return jr.ctx.vm.ToValue(int(ret))
}

// botFinalTask(bot:FinalTask(name, ...args) -> RetVal)
func (jr *jsBot) botFinalTask(call goja.FunctionCall) goja.Value {
	const methodName = "FinalTask"

	// Argument #0: name (string)
	name := jr.requireStringArg(methodName, call, 0)

	if name == "" {
		panic(jr.ctx.vm.ToValue("FinalTask: name must not be empty"))
	}

	// Arguments #1..N: args (strings)
	extras := parseStringArgs(jr.ctx.vm, call, 1)

	// Call the underlying Go method
	ret := jr.r.FinalTask(name, extras...)

	// Return the RetVal as an integer
	return jr.ctx.vm.ToValue(int(ret))
}

// botFailTask(bot:FailTask(name, ...args) -> RetVal)
func (jr *jsBot) botFailTask(call goja.FunctionCall) goja.Value {
	const methodName = "FailTask"

	// Argument #0: name (string)
	name := jr.requireStringArg(methodName, call, 0)

	if name == "" {
		panic(jr.ctx.vm.ToValue("FailTask: name must not be empty"))
	}

	// Arguments #1..N: args (strings)
	extras := parseStringArgs(jr.ctx.vm, call, 1)

	// Call the underlying Go method
	ret := jr.r.FailTask(name, extras...)

	// Return the RetVal as an integer
	return jr.ctx.vm.ToValue(int(ret))
}

// botAddJob(bot:AddJob(name, ...args) -> RetVal)
func (jr *jsBot) botAddJob(call goja.FunctionCall) goja.Value {
	const methodName = "AddJob"

	// Argument #0: name (string)
	name := jr.requireStringArg(methodName, call, 0)

	if name == "" {
		panic(jr.ctx.vm.ToValue("AddJob: name must not be empty"))
	}

	// Arguments #1..N: args (strings)
	extras := parseStringArgs(jr.ctx.vm, call, 1)

	// Call the underlying Go method
	ret := jr.r.AddJob(name, extras...)

	// Return the RetVal as an integer
	return jr.ctx.vm.ToValue(int(ret))
}

// botAddCommand(bot:AddCommand(pluginName, command) -> RetVal)
func (jr *jsBot) botAddCommand(call goja.FunctionCall) goja.Value {
	const methodName = "AddCommand"

	// Argument #0: pluginName (string)
	pluginName := jr.requireStringArg(methodName, call, 0)

	// Argument #1: command (string)
	command := jr.requireStringArg(methodName, call, 1)

	if pluginName == "" {
		panic(jr.ctx.vm.ToValue("AddCommand: pluginName must not be empty"))
	}
	if command == "" {
		panic(jr.ctx.vm.ToValue("AddCommand: command must not be empty"))
	}

	// Call the underlying Go method
	ret := jr.r.AddCommand(pluginName, command)

	// Return the RetVal as an integer
	return jr.ctx.vm.ToValue(int(ret))
}

// botFinalCommand(bot:FinalCommand(pluginName, command) -> RetVal)
func (jr *jsBot) botFinalCommand(call goja.FunctionCall) goja.Value {
	const methodName = "FinalCommand"

	// Argument #0: pluginName (string)
	pluginName := jr.requireStringArg(methodName, call, 0)

	// Argument #1: command (string)
	command := jr.requireStringArg(methodName, call, 1)

	if pluginName == "" {
		panic(jr.ctx.vm.ToValue("FinalCommand: pluginName must not be empty"))
	}
	if command == "" {
		panic(jr.ctx.vm.ToValue("FinalCommand: command must not be empty"))
	}

	// Call the underlying Go method
	ret := jr.r.FinalCommand(pluginName, command)

	// Return the RetVal as an integer
	return jr.ctx.vm.ToValue(int(ret))
}

// botFailCommand(bot:FailCommand(pluginName, command) -> RetVal)
func (jr *jsBot) botFailCommand(call goja.FunctionCall) goja.Value {
	const methodName = "FailCommand"

	// Argument #0: pluginName (string)
	pluginName := jr.requireStringArg(methodName, call, 0)

	// Argument #1: command (string)
	command := jr.requireStringArg(methodName, call, 1)

	if pluginName == "" {
		panic(jr.ctx.vm.ToValue("FailCommand: pluginName must not be empty"))
	}
	if command == "" {
		panic(jr.ctx.vm.ToValue("FailCommand: command must not be empty"))
	}

	// Call the underlying Go method
	ret := jr.r.FailCommand(pluginName, command)

	// Return the RetVal as an integer
	return jr.ctx.vm.ToValue(int(ret))
}

// parseStringArgs extracts string arguments starting from a given index.
// It returns a slice of strings. If any argument is not a string, it panics.
func parseStringArgs(rt *goja.Runtime, call goja.FunctionCall, start int) []string {
	args := []string{}
	for i := start; i < len(call.Arguments); i++ {
		argVal := call.Arguments[i].Export()
		str, ok := argVal.(string)
		if !ok {
			panic(rt.ToValue(fmt.Sprintf(
				"Expected argument #%d to be a string, got %T",
				i+1, argVal,
			)))
		}
		args = append(args, str)
	}
	return args
}
