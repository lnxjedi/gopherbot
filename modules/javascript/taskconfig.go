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
	cfgObject := jr.ctx.vm.NewObject()

	mapConfig := make(map[string]interface{})
	retVal := jr.r.GetTaskConfig(&mapConfig)
	if retVal == robot.Ok {
		cfgObject.Set("retVal", retVal)
		configVal, err := parseGoValueToJS(jr.ctx.vm, mapConfig)
		if err != nil {
			cfgObject.Set("retVal", robot.ConfigUnmarshalError)
			cfgObject.Set("config", goja.Null())
			// This is unusual (bad JSON?), log and return ConfigUnmarshalError
			jr.log(robot.Error, fmt.Sprintf("Error converting map config to JS: %v", err))
			return cfgObject
		}
		cfgObject.Set("config", configVal)
		return cfgObject
	}
	if retVal == robot.ConfigUnmarshalError {
		var sliceConfig []interface{}
		retVal = jr.r.GetTaskConfig(&sliceConfig)
		if retVal == robot.Ok {
			cfgObject.Set("retVal", retVal)
			configVal, err := parseGoValueToJS(jr.ctx.vm, sliceConfig)
			if err != nil {
				cfgObject.Set("retVal", robot.ConfigUnmarshalError)
				cfgObject.Set("config", goja.Null())
				// This is unusual (bad JSON?), log and return ConfigUnmarshalError
				jr.log(robot.Error, fmt.Sprintf("Error converting slice config to JS: %v", err))
				return cfgObject
			}
			cfgObject.Set("config", configVal)
			return cfgObject
		}
	}
	cfgObject.Set("retVal", retVal)
	cfgObject.Set("config", goja.Null())
	return cfgObject
}
