// shortterm_memories.go
package javascript

import (
	"fmt"

	"github.com/dop251/goja"
)

// -------------------------------------------------------------------
// 1. bot:Remember(key, value, shared)
// -------------------------------------------------------------------

// botRemember allows JavaScript scripts to remember a key-value pair with an optional shared flag.
func (jr *jsBot) botRemember(call goja.FunctionCall) goja.Value {
	const methodName = "Remember"

	// Argument #0: key (string)
	key := jr.requireStringArg(methodName, call, 0)

	// Argument #1: value (string)
	value := jr.requireStringArg(methodName, call, 1)

	// Argument #2: shared (boolean, optional, default false)
	shared := false
	if len(call.Arguments) > 2 {
		rawVal := call.Arguments[2].Export()
		if b, ok := rawVal.(bool); ok {
			shared = b
		} else if rawVal != nil {
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: invalid value for 'shared' (must be boolean or omitted)",
				methodName,
			)))
		}
	}

	// Validate non-empty key and value
	if key == "" {
		panic(jr.ctx.vm.ToValue("Remember: key must not be empty"))
	}
	if value == "" {
		panic(jr.ctx.vm.ToValue("Remember: value must not be empty"))
	}

	// Call the underlying Go method
	jr.r.Remember(key, value, shared)

	// Return true to indicate success
	return jr.ctx.vm.ToValue(true)
}

// -------------------------------------------------------------------
// 2. bot:RememberThread(key, value, shared)
// -------------------------------------------------------------------

// botRememberThread remembers a key-value pair in a threaded context with an optional shared flag.
func (jr *jsBot) botRememberThread(call goja.FunctionCall) goja.Value {
	const methodName = "RememberThread"

	// Argument #0: key (string)
	key := jr.requireStringArg(methodName, call, 0)

	// Argument #1: value (string)
	value := jr.requireStringArg(methodName, call, 1)

	// Argument #2: shared (boolean, optional, default false)
	shared := false
	if len(call.Arguments) > 2 {
		rawVal := call.Arguments[2].Export()
		if b, ok := rawVal.(bool); ok {
			shared = b
		} else if rawVal != nil {
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: invalid value for 'shared' (must be boolean or omitted)",
				methodName,
			)))
		}
	}

	// Validate non-empty key and value
	if key == "" {
		panic(jr.ctx.vm.ToValue("RememberThread: key must not be empty"))
	}
	if value == "" {
		panic(jr.ctx.vm.ToValue("RememberThread: value must not be empty"))
	}

	// Call the underlying Go method
	jr.r.RememberThread(key, value, shared)

	// Return true to indicate success
	return jr.ctx.vm.ToValue(true)
}

// -------------------------------------------------------------------
// 3. bot:RememberContext(context, value)
// -------------------------------------------------------------------

// botRememberContext remembers a value within a specific context.
func (jr *jsBot) botRememberContext(call goja.FunctionCall) goja.Value {
	const methodName = "RememberContext"

	// Argument #0: context (string)
	context := jr.requireStringArg(methodName, call, 0)

	// Argument #1: value (string)
	value := jr.requireStringArg(methodName, call, 1)

	// Validate non-empty context and value
	if context == "" {
		panic(jr.ctx.vm.ToValue("RememberContext: context must not be empty"))
	}
	if value == "" {
		panic(jr.ctx.vm.ToValue("RememberContext: value must not be empty"))
	}

	// Call the underlying Go method
	jr.r.RememberContext(context, value)

	// Return true to indicate success
	return jr.ctx.vm.ToValue(true)
}

// -------------------------------------------------------------------
// 4. bot:RememberContextThread(context, value)
// -------------------------------------------------------------------

// botRememberContextThread remembers a value within a specific context in a threaded environment.
func (jr *jsBot) botRememberContextThread(call goja.FunctionCall) goja.Value {
	const methodName = "RememberContextThread"

	// Argument #0: context (string)
	context := jr.requireStringArg(methodName, call, 0)

	// Argument #1: value (string)
	value := jr.requireStringArg(methodName, call, 1)

	// Validate non-empty context and value
	if context == "" {
		panic(jr.ctx.vm.ToValue("RememberContextThread: context must not be empty"))
	}
	if value == "" {
		panic(jr.ctx.vm.ToValue("RememberContextThread: value must not be empty"))
	}

	// Call the underlying Go method
	jr.r.RememberContextThread(context, value)

	// Return true to indicate success
	return jr.ctx.vm.ToValue(true)
}

// -------------------------------------------------------------------
// 5. bot:Recall(key, shared) -> string
// -------------------------------------------------------------------

// botRecall recalls a value by key with an optional shared flag.
func (jr *jsBot) botRecall(call goja.FunctionCall) goja.Value {
	const methodName = "Recall"

	// Argument #0: key (string)
	key := jr.requireStringArg(methodName, call, 0)

	// Argument #1: shared (boolean, optional, default false)
	shared := false
	if len(call.Arguments) > 1 {
		rawVal := call.Arguments[1].Export()
		if b, ok := rawVal.(bool); ok {
			shared = b
		} else if rawVal != nil {
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: invalid value for 'shared' (must be boolean or omitted)",
				methodName,
			)))
		}
	}

	// Validate non-empty key
	if key == "" {
		panic(jr.ctx.vm.ToValue("Recall: key must not be empty"))
	}

	// Call the underlying Go method
	value := jr.r.Recall(key, shared)

	// Return the recalled value as a string
	return jr.ctx.vm.ToValue(value)
}
