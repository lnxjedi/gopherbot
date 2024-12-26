// call_extension.go
package javascript

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/lnxjedi/gopherbot/robot"
)

// jsContext holds a reference to the robot.Robot interface, the environment fields,
// and the goja.Runtime we'll execute the script in.
type jsContext struct {
	r            robot.Robot
	env          map[string]string
	vm           *goja.Runtime
	requirePaths []string
}

// CallExtension loads and executes a JavaScript file with goja:
//   - taskPath, taskName - the path to script and its name
//   - pkgPath - directories the script should search for requires
//   - env - env vars normally passed to external scripts, has thread info
//   - r: the robot.Robot
//   - args: the script arguments
func CallExtension(taskPath, taskName string, requirePaths []string, env map[string]string, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	// Create a new goja VM
	vm := goja.New()

	// Add the simple http interface
	addHttpHandler(vm)

	ctx := &jsContext{
		r:            r,
		env:          env,
		vm:           vm,
		requirePaths: requirePaths,
	}

	// Stub for adding additional require paths or preloading modules
	// (like the Lua version's updatePkgPath). The user will implement it.
	retVal, err := ctx.addRequires(vm)
	if err != nil {
		return retVal, err
	}

	// Expose the "robot" object in JS with a .New() method returning a "bot" object
	// Also set up standard constants from gopherbot
	ctx.registerRobotObject()

	// Provide the script arguments as an array "global.argv"
	ctx.setArgv(taskName, args)

	// Read and run the JS file
	scriptBytes, err := os.ReadFile(taskPath)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to read JS file '%s': %w", taskPath, err)
	}

	program, compileErr := goja.Compile(taskName, string(scriptBytes), true)
	if compileErr != nil {
		return robot.MechanismFail, fmt.Errorf("JavaScript compile error in '%s': %w", taskName, compileErr)
	}

	ret, runErr := ctx.runProgram(program)
	if runErr != nil {
		return robot.MechanismFail, fmt.Errorf("JavaScript runtime error in '%s': %w", taskName, runErr)
	}
	return ret, nil
}

// runProgram runs a compiled *goja.Program and returns the robot.TaskRetVal.
func (ctx *jsContext) runProgram(prog *goja.Program) (robot.TaskRetVal, error) {
	ret, err := ctx.vm.RunProgram(prog)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("JavaScript runtime error: %w", err)
	}

	// If the script didn't return a value, it's considered Normal
	if ret == nil || ret == goja.Undefined() || ret == goja.Null() {
		return robot.Normal, nil
	}

	// Attempt to convert the return value to a robot.TaskRetVal (int)
	if retInt, ok := ret.Export().(int); ok {
		return robot.TaskRetVal(retInt), nil
	} else if retInt, ok := ret.Export().(int64); ok {
		return robot.TaskRetVal(int(retInt)), nil
	} else if retFloat, ok := ret.Export().(float64); ok {
		return robot.TaskRetVal(int(retFloat)), nil
	} else if retBool, ok := ret.Export().(bool); ok {
		if retBool {
			return robot.Normal, nil
		} else {
			return robot.Fail, nil
		}
	}

	// Handle other return types or if the type assertion fails
	return robot.MechanismFail, fmt.Errorf("JavaScript script did not return a valid status (int, bool, or nil)")
}

// setArgv creates the global "argv" array in JS so scripts can read arguments
// similarly to process.argv in Node.
func (ctx *jsContext) setArgv(taskName string, args []string) {
	// e.g. argv[0] = "taskName"
	//      argv[1..n] = the rest
	argv := make([]interface{}, 0, len(args)+1)
	argv = append(argv, taskName)
	for _, a := range args {
		argv = append(argv, a)
	}
	ctx.vm.Set("argv", argv)
}

// addRequires sets up a require() function using goja_nodejs, allowing JavaScript
// scripts to load other scripts/modules from the given paths.
func (ctx *jsContext) addRequires(vm *goja.Runtime) (robot.TaskRetVal, error) {
	registry := require.NewRegistry(
		require.WithGlobalFolders(ctx.requirePaths...),
	)

	if err := registry.Enable(vm); err != nil {
		return robot.Fail, fmt.Errorf("failed to enable 'require' for javascript: %w", err)
	}

	return robot.Normal, nil
}
