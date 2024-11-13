package yaegidynamicgo

import (
	"fmt"
	"path/filepath"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// GetPluginHandler loads and compiles a Go plugin from the given path using Yaegi,
// retrieves the PluginHandler function, and returns it for execution.
func GetPluginHandler(path string) (func(r robot.Robot, command string, args ...string) robot.TaskRetVal, error) {
	// Initialize the interpreter with necessary configurations
	i, err := setupRobotInterpreter()
	if err != nil {
		return nil, fmt.Errorf("interpreter setup failed: %w", err)
	}

	// Compile and run the plugin source code
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin path: %w", err)
	}
	_, err = i.CompilePath(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compile plugin: %w", err)
	}

	// Retrieve the PluginHandler symbol
	v, err := i.Eval("PluginHandler")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve PluginHandler: %w", err)
	}

	// Assert that PluginHandler has the correct signature
	handler, ok := v.Interface().(func(robot.Robot, string, ...string) robot.TaskRetVal)
	if !ok {
		return nil, fmt.Errorf("PluginHandler has incorrect signature")
	}

	return handler, nil
}

// setupRobotInterpreter initializes a Yaegi interpreter with the robot package symbols
// and the standard library.
func setupRobotInterpreter() (*interp.Interpreter, error) {
	i := interp.New(interp.Options{})

	// Load the standard library
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, fmt.Errorf("failed to load standard library: %w", err)
	}

	// Load the robot package symbols
	if err := i.Use(robot.Symbols); err != nil {
		return nil, fmt.Errorf("failed to load robot symbols: %w", err)
	}

	return i, nil
}
