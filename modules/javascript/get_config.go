// get_config.go
package javascript

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
)

// GetPluginConfig calls the given JS script with the argument "configure".
// We expect the script to return a YAML string that we convert to *[]byte.
func GetPluginConfig(execPath, taskPath, taskName string, emptyBot map[string]string, libPaths []string) (*[]byte, error) {
	vm := goja.New()

	ctx := &jsContext{
		r:            nil,
		bot:          emptyBot,
		vm:           vm,
		requirePaths: libPaths,
	}

	ctx.addRequires(vm)

	// Create a "process" object with an "argv" array
	processObj := vm.NewObject()
	processObj.Set("argv", []string{execPath, taskPath, "configure"}) // Add a dummy value at index 0

	// Set the "process" object as a global variable
	vm.Set("process", processObj)

	scriptBytes, err := os.ReadFile(taskPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JS config script '%s': %w", taskName, err)
	}

	prog, compileErr := goja.Compile(taskName, string(scriptBytes), true)
	if compileErr != nil {
		return nil, fmt.Errorf("JavaScript compile error in '%s': %w", taskName, compileErr)
	}

	cfg, runErr := vm.RunProgram(prog) // Capture the return value
	if runErr != nil {
		if ex, ok := runErr.(*goja.Exception); ok {
			return nil, fmt.Errorf("Javascript exception from %s: %s", taskName, ex.String())
		}
		return nil, fmt.Errorf("JavaScript runtime error in '%s': %w", taskName, runErr)
	}

	// Check if the return value is a string
	cfgStr, ok := cfg.Export().(string)
	if !ok {
		return nil, fmt.Errorf("JavaScript plugin %s did not return a string for 'configure'", taskName)
	}

	cfgBytes := []byte(cfgStr)
	return &cfgBytes, nil
}
