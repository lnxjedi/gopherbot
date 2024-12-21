package lua

import (
	"fmt"

	glua "github.com/yuin/gopher-lua"
)

// GetPluginConfig calls the given Lua script with the argument "configure".
// We expect the script to return a YAML string that we convert to *[]byte.
func GetPluginConfig(taskPath, taskName string) (*[]byte, error) {
	L := glua.NewState()
	defer L.Close()
	// For now, load all standard libs
	L.OpenLibs()

	// Create the args global with a single element: "configure"
	argsTable := L.CreateTable(1, 0)
	argsTable.RawSetInt(1, glua.LString("configure"))
	L.SetGlobal("args", argsTable)

	// Load + Run the script
	if err := L.DoFile(taskPath); err != nil {
		return nil, fmt.Errorf("error loading Lua config for %s: %v", taskName, err)
	}

	// The script's return should be a single value on the stack (the config string).
	retVal := L.Get(-1) // top of stack
	L.Pop(1)            // pop it

	cfgStr, ok := retVal.(glua.LString)
	if !ok {
		// If the script didn't return a string, that’s an error
		return nil, fmt.Errorf("Lua plugin %s did not return a YAML string from 'configure'", taskName)
	}

	// Convert to []byte
	cfgBytes := []byte(cfgStr)

	return &cfgBytes, nil
}
