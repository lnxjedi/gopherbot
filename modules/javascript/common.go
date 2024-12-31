package javascript

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// setProcessArgv creates the global "process.argv" array in JS so scripts can read arguments
// similarly to process.argv in Node.
func (ctx *jsContext) setProcessArgv(execPath, taskPath string, args ...string) {
	// e.g. process.argv[0] = "/path/to/gopherbot"
	//      process.argv[1] = "/path/to/script.js"
	//      process.argv[2..n] = the rest
	argv := make([]interface{}, 0, len(args)+2)

	argv = append(argv, execPath) // argv[0]: The binary name
	argv = append(argv, taskPath) // argv[1]: The script name

	for _, a := range args {
		argv = append(argv, a)
	}

	// Create the global "process" object if it doesn't exist
	if obj := ctx.vm.Get("process"); obj == nil || obj.StrictEquals(goja.Undefined()) {
		ctx.vm.Set("process", ctx.vm.NewObject())
	}

	// Get the "process" object
	processObj := ctx.vm.Get("process").ToObject(ctx.vm)

	// Set the "argv" property on the "process" object
	processObj.Set("argv", argv)
}

// addRequires sets up a require() function using goja_nodejs, allowing JavaScript
// scripts to load other scripts/modules from the given paths.
func (ctx *jsContext) addRequires(vm *goja.Runtime) {
	registry := require.NewRegistry(
		require.WithGlobalFolders(ctx.requirePaths...),
	)

	registry.Enable(vm)
}

// requireStringArg generates a js exception if we didn't get a string argument
func (ctx *jsContext) requireStringArg(methodName string, call goja.FunctionCall, index int) string {
	// Make sure we actually have enough arguments
	if len(call.Arguments) <= index {
		panic(ctx.vm.ToValue(fmt.Sprintf(
			"%s: missing argument #%d", methodName, index+1,
		)))
	}

	// Export the goja.Value to a Go interface{}
	rawVal := call.Arguments[index].Export()

	// Try asserting that interface{} is a string
	s, ok := rawVal.(string)
	if !ok {
		panic(ctx.vm.ToValue(fmt.Sprintf(
			"%s: argument #%d must be a string, got %T",
			methodName, index+1, rawVal,
		)))
	}

	return s
}
