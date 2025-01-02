package javascript

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

// botGetTaskConfig(bot:GetTaskConfig()) -> { config, retVal }
//
// Usage in JS:
//
//	let result = bot.GetTaskConfig();
//	if(result.retVal === ret.Ok) {
//	    console.log("Config object:", result.config);
//	} else {
//	    console.log("Error code:", result.retVal);
//	}
func (jr *jsBot) botGetTaskConfig(call goja.FunctionCall) goja.Value {
	const methodName = "GetTaskConfig"

	// We expect no arguments here, but you could do a check if desired:
	// if len(call.Arguments) > 0 { ... }

	cfgObject := jr.ctx.vm.NewObject()

	var config interface{}
	retVal := jr.r.GetTaskConfig(&config)
	cfgObject.Set("retVal", retVal)
	if retVal == robot.Ok {
		configVal, err := parseGoValueToJS(jr.ctx.vm, config)
		if err != nil {
			cfgObject.Set("config", goja.Null())
			// This is unusual (bad JSON?), log and return DataFormatError
			jr.log(robot.Error, fmt.Sprintf("Error converting map config to JS: %v", err))
			return cfgObject
		}
		cfgObject.Set("config", configVal)
		return cfgObject
	}
	cfgObject.Set("config", goja.Null())
	return cfgObject
}
