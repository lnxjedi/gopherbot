// call_extension.go
package javascript

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

// jsContext holds a reference to the robot.Robot interface, the environment fields,
// and the goja.Runtime we'll execute the script in.
type jsContext struct {
	r   robot.Robot
	env map[string]string
	vm  *goja.Runtime
}

// CallExtension loads and executes a JavaScript file with goja:
//   - taskPath, taskName - the path to script and its name
//   - pkgPath - directories the script should search for requires
//   - env - env vars normally passed to external scripts, has thread info
//   - r: the robot.Robot
//   - args: the script arguments
func CallExtension(taskPath, taskName string, pkgPath []string, env map[string]string, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	// Create a new goja VM
	vm := goja.New()

	ctx := &jsContext{
		r:   r,
		env: env,
		vm:  vm,
	}

	// Stub for adding additional require paths or preloading modules
	// (like the Lua version's updatePkgPath). The user will implement it.
	retVal, err := addRequires(vm, r, pkgPath)
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

	runErr := ctx.runProgram(program)
	if runErr != nil {
		return robot.MechanismFail, fmt.Errorf("JavaScript runtime error in '%s': %w", taskName, runErr)
	}

	// The script's return value is on top of the VM stack as the last value
	// We treat it as a number or string if possible, default to Normal.
	val := vm.Get("exports") // By convention, or you can do a final "return" from script
	if val == nil || val == goja.Undefined() || val == goja.Null() {
		return robot.Normal, nil
	}

	switch ret := val.Export().(type) {
	case int64:
		return robot.TaskRetVal(ret), nil
	case float64:
		return robot.TaskRetVal(int64(ret)), nil
	case string:
		// If you return a string, that’s not necessarily a TaskRetVal.
		// Fallback to Normal or parse if you like.
		r.Log(robot.Debug, fmt.Sprintf("JS script returned a string: %s", ret))
		return robot.Normal, nil
	default:
		return robot.Normal, nil
	}
}

// runProgram runs a compiled *goja.Program.
func (ctx *jsContext) runProgram(prog *goja.Program) error {
	_, err := ctx.vm.RunProgram(prog)
	return err
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

// addRequires is a stub for the user to fill in, analogous to updatePkgPath in Lua.
func addRequires(vm *goja.Runtime, r robot.Robot, pkgPath []string) (robot.TaskRetVal, error) {
	// The user can implement custom require() logic, e.g. hooking into goja's Resolve, etc.
	// Or add global / preloaded modules from these pkgPaths.
	//
	// NOTE: This is a placeholder. Return success by default.
	return robot.Normal, nil
}
