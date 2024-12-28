package lua

import (
	"fmt"

	glua "github.com/yuin/gopher-lua"
)

// GetPluginConfig calls the given Lua script with the argument "configure".
// We expect the script to return a YAML string that we convert to *[]byte.
func GetPluginConfig(execPath, taskPath, taskName string, emptyBot map[string]string, pkgPath []string) (*[]byte, error) {
	L := glua.NewState()
	defer L.Close()

	// Initialize the luaRobot with fields from the bot map
	botFields, err := initializeFields(emptyBot)
	if err != nil {
		return nil, err
	}

	// Add the Lua arg table for "configure"
	addArgTable(L, execPath, taskPath, "configure")

	// Modify OS functions to replace os.setenv and os.setlocale with no-ops
	modifyOSFunctions(L, nil)

	// Well, the FIRST time it's called it definitely will - but not in all the
	// component Register* functions below ...
	registerBotMetatableIfNeeded(L)

	// We don't register API methods for GetPluginConfig

	// Create the primary robot userdata and set it as "robot"
	robotUD := newLuaBot(L, nil, botFields)
	L.SetGlobal("GBOT", robotUD)

	// **Update package.path with additional directories and Lua patterns**
	_, err = updatePkgPath(L, nil, pkgPath)
	if err != nil {
		return nil, err
	}

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
