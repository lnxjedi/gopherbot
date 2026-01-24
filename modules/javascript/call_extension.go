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
	r            robot.Robot
	l            robot.Logger
	bot          map[string]string
	vm           *goja.Runtime
	requirePaths []string
}

// CallExtension loads and executes a JavaScript file with goja:
//   - taskPath, taskName - the path to script and its name
//   - pkgPath - directories the script should search for requires
//   - env - env vars normally passed to external scripts, has thread info
//   - r: the robot.Robot
//   - args: the script arguments
func CallExtension(execPath, taskPath, taskName string, requirePaths []string, logger robot.Logger,
	realBot map[string]string, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	// Create a new goja VM
	vm := goja.New()

	ctx := &jsContext{
		r:            r,
		l:            logger,
		bot:          realBot,
		vm:           vm,
		requirePaths: requirePaths,
	}

	ctx.addRequires(vm)

	// Create the first robot object and the global "GBOT"
	firstRobot := &jsBot{
		r:   r,
		ctx: ctx,
	}
	firstBotObj := firstRobot.createBotObject()
	vm.Set("GBOT", firstBotObj)

	err := ctx.setProcessArgv(execPath, taskPath, args...)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to set args: %w", err)
	}

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
