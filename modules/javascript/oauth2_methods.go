package javascript

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

func (jr *jsBot) botGetIdentityCredential(call goja.FunctionCall) goja.Value {
	const methodName = "GetIdentityCredential"
	provider := jr.requireStringArg(methodName, call, 0)
	user := jr.requireStringArg(methodName, call, 1)
	credential, retVal := jr.r.GetIdentityCredential(provider, user)
	resultObj := jr.ctx.vm.NewObject()
	if credential != nil {
		credentialObj := jr.ctx.vm.NewObject()
		credentialObj.Set("type", credential.Type)
		credentialObj.Set("value", credential.Value)
		credentialObj.Set("scheme", credential.Scheme)
		credentialObj.Set("headerName", credential.HeaderName)
		credentialObj.Set("headerValue", credential.HeaderValue)
		credentialObj.Set("expiresAt", credential.ExpiresAt)
		credentialObj.Set("metadata", credential.Metadata)
		resultObj.Set("credential", credentialObj)
	} else {
		resultObj.Set("credential", nil)
	}
	resultObj.Set("retVal", int(retVal))
	return resultObj
}

func (jr *jsBot) botLinkOAuth2Identity(call goja.FunctionCall) goja.Value {
	const methodName = "LinkOAuth2Identity"
	if len(call.Arguments) < 1 {
		panic(jr.ctx.vm.ToValue(fmt.Sprintf("%s: requires a link object", methodName)))
	}
	exported := call.Arguments[0].Export()
	blob, err := json.Marshal(exported)
	if err != nil {
		return jr.ctx.vm.ToValue(int(robot.IdentityInvalidLinkRequest))
	}
	var req robot.OAuth2IdentityLinkRequest
	if err := json.Unmarshal(blob, &req); err != nil {
		return jr.ctx.vm.ToValue(int(robot.IdentityInvalidLinkRequest))
	}
	return jr.ctx.vm.ToValue(int(jr.r.LinkOAuth2Identity(&req)))
}

func (jr *jsBot) botUnlinkIdentity(call goja.FunctionCall) goja.Value {
	const methodName = "UnlinkIdentity"
	provider := jr.requireStringArg(methodName, call, 0)
	user := jr.requireStringArg(methodName, call, 1)
	return jr.ctx.vm.ToValue(int(jr.r.UnlinkIdentity(provider, user)))
}
