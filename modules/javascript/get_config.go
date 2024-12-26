// get_config.go
package javascript

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
)

// GetPluginConfig calls the given JS script with the argument "configure".
// We expect the script to return a YAML string that we convert to *[]byte.
func GetPluginConfig(taskPath, taskName string, pkgPath []string) (*[]byte, error) {
	vm := goja.New()

	// Stub for adding requires. We don’t have a robot here, pass nil if needed.
	_, err := addRequires(vm, nil, pkgPath)
	if err != nil {
		return nil, err
	}

	// Simulate `argv = [ taskName, "configure" ]`
	vm.Set("argv", []interface{}{taskName, "configure"})

	scriptBytes, err := os.ReadFile(taskPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JS config script '%s': %w", taskName, err)
	}

	prog, compileErr := goja.Compile(taskName, string(scriptBytes), true)
	if compileErr != nil {
		return nil, fmt.Errorf("JavaScript compile error in '%s': %w", taskName, compileErr)
	}

	_, runErr := vm.RunProgram(prog)
	if runErr != nil {
		return nil, fmt.Errorf("JavaScript runtime error in '%s': %w", taskName, runErr)
	}

	// The script should return the config as a string, similar to the Lua version
	val := vm.Get("exports") // or however you want to retrieve the "returned" value
	if val == nil || val == goja.Undefined() || val == goja.Null() {
		return nil, fmt.Errorf("JavaScript plugin %s did not return a YAML string from 'configure'", taskName)
	}

	cfgStr, ok := val.Export().(string)
	if !ok {
		return nil, fmt.Errorf("JavaScript plugin %s did not return a string for 'configure'", taskName)
	}

	cfgBytes := []byte(cfgStr)
	return &cfgBytes, nil
}
