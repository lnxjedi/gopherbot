package javascript

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

func (jr *jsBot) botGetOAuth2Token(call goja.FunctionCall) goja.Value {
	const methodName = "GetOAuth2Token"
	provider := jr.requireStringArg(methodName, call, 0)
	user := jr.requireStringArg(methodName, call, 1)
	token, retVal := jr.r.GetOAuth2Token(provider, user)
	resultObj := jr.ctx.vm.NewObject()
	resultObj.Set("token", token)
	resultObj.Set("retVal", int(retVal))
	return resultObj
}

func (jr *jsBot) botLinkOAuth2User(call goja.FunctionCall) goja.Value {
	const methodName = "LinkOAuth2User"
	if len(call.Arguments) < 1 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s: requires a link object", methodName)))
	}
	exported := call.Arguments[0].Export()
	blob, err := json.Marshal(exported)
	if err != nil {
		return jr.ctx.vm.ToValue(int(robot.OAuth2InvalidLinkRequest))
	}
	var req robot.OAuth2LinkRequest
	if err := json.Unmarshal(blob, &req); err != nil {
		return jr.ctx.vm.ToValue(int(robot.OAuth2InvalidLinkRequest))
	}
	return jr.ctx.vm.ToValue(int(jr.r.LinkOAuth2User(&req)))
}

func (jr *jsBot) botUnlinkOAuth2User(call goja.FunctionCall) goja.Value {
	const methodName = "UnlinkOAuth2User"
	provider := jr.requireStringArg(methodName, call, 0)
	user := jr.requireStringArg(methodName, call, 1)
	return jr.ctx.vm.ToValue(int(jr.r.UnlinkOAuth2User(provider, user)))
}
