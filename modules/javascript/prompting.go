package javascript

import (
	"fmt"

	"github.com/dop251/goja"
)

// botPromptForReply(bot:PromptForReply(regexID, prompt?))
//
// JavaScript usage example:
//
//	let result = bot.PromptForReply("someRegexID", "Please reply");
//	console.log(`Reply: ${result.reply}, retVal: ${result.retVal}`);
func (jr *jsBot) botPromptForReply(call goja.FunctionCall) goja.Value {
	const methodName = "PromptForReply"

	// Argument #0: regexID (string)
	regexID := jr.requireStringArg(methodName, call, 0)
	if regexID == "" {
		panic(jr.ctx.vm.ToValue("PromptForReply: regexID must not be empty"))
	}

	// Argument #1: prompt (string), optional, defaults to ""
	var prompt string
	if len(call.Arguments) > 1 {
		rawPrompt := call.Arguments[1].Export()
		if str, ok := rawPrompt.(string); ok {
			prompt = str
		} else {
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: argument #2 (prompt) must be a string if provided",
				methodName,
			)))
		}
	}

	reply, ret := jr.r.PromptForReply(regexID, prompt)

	// Return an object with { reply, retVal }
	res := jr.ctx.vm.NewObject()
	res.Set("reply", reply)
	res.Set("retVal", int(ret))
	return res
}

// botPromptThreadForReply(bot:PromptThreadForReply(regexID, prompt?))
func (jr *jsBot) botPromptThreadForReply(call goja.FunctionCall) goja.Value {
	const methodName = "PromptThreadForReply"

	// Argument #0: regexID (string)
	regexID := jr.requireStringArg(methodName, call, 0)
	if regexID == "" {
		panic(jr.ctx.vm.ToValue("PromptThreadForReply: regexID must not be empty"))
	}

	// Argument #1: prompt (string), optional
	var prompt string
	if len(call.Arguments) > 1 {
		rawPrompt := call.Arguments[1].Export()
		if str, ok := rawPrompt.(string); ok {
			prompt = str
		} else {
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: argument #2 (prompt) must be a string if provided",
				methodName,
			)))
		}
	}

	reply, ret := jr.r.PromptThreadForReply(regexID, prompt)

	res := jr.ctx.vm.NewObject()
	res.Set("reply", reply)
	res.Set("retVal", int(ret))
	return res
}

// botPromptUserForReply(bot:PromptUserForReply(regexID, user, prompt?))
func (jr *jsBot) botPromptUserForReply(call goja.FunctionCall) goja.Value {
	const methodName = "PromptUserForReply"

	// Argument #0: regexID (string)
	regexID := jr.requireStringArg(methodName, call, 0)
	if regexID == "" {
		panic(jr.ctx.vm.ToValue("PromptUserForReply: regexID must not be empty"))
	}

	// Argument #1: user (string)
	user := jr.requireStringArg(methodName, call, 1)
	if user == "" {
		panic(jr.ctx.vm.ToValue("PromptUserForReply: user must not be empty"))
	}

	// Argument #2: prompt (string), optional
	var prompt string
	if len(call.Arguments) > 2 {
		rawPrompt := call.Arguments[2].Export()
		if str, ok := rawPrompt.(string); ok {
			prompt = str
		} else {
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: argument #3 (prompt) must be a string if provided",
				methodName,
			)))
		}
	}

	reply, ret := jr.r.PromptUserForReply(regexID, user, prompt)

	res := jr.ctx.vm.NewObject()
	res.Set("reply", reply)
	res.Set("retVal", int(ret))
	return res
}

// botPromptUserChannelForReply(bot:PromptUserChannelForReply(regexID, user, channel, prompt?))
func (jr *jsBot) botPromptUserChannelForReply(call goja.FunctionCall) goja.Value {
	const methodName = "PromptUserChannelForReply"

	// Argument #0: regexID (string)
	regexID := jr.requireStringArg(methodName, call, 0)
	if regexID == "" {
		panic(jr.ctx.vm.ToValue("PromptUserChannelForReply: regexID must not be empty"))
	}

	// Argument #1: user (string)
	user := jr.requireStringArg(methodName, call, 1)
	if user == "" {
		panic(jr.ctx.vm.ToValue("PromptUserChannelForReply: user must not be empty"))
	}

	// Argument #2: channel (string)
	channel := jr.requireStringArg(methodName, call, 2)
	if channel == "" {
		panic(jr.ctx.vm.ToValue("PromptUserChannelForReply: channel must not be empty"))
	}

	// Argument #3: prompt (string), optional
	var prompt string
	if len(call.Arguments) > 3 {
		rawPrompt := call.Arguments[3].Export()
		if str, ok := rawPrompt.(string); ok {
			prompt = str
		} else {
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: argument #4 (prompt) must be a string if provided",
				methodName,
			)))
		}
	}

	reply, ret := jr.r.PromptUserChannelForReply(regexID, user, channel, prompt)

	res := jr.ctx.vm.NewObject()
	res.Set("reply", reply)
	res.Set("retVal", int(ret))
	return res
}

// botPromptUserChannelThreadForReply(bot:PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt?))
func (jr *jsBot) botPromptUserChannelThreadForReply(call goja.FunctionCall) goja.Value {
	const methodName = "PromptUserChannelThreadForReply"

	// Argument #0: regexID (string)
	regexID := jr.requireStringArg(methodName, call, 0)
	if regexID == "" {
		panic(jr.ctx.vm.ToValue("PromptUserChannelThreadForReply: regexID must not be empty"))
	}

	// Argument #1: user (string)
	user := jr.requireStringArg(methodName, call, 1)
	if user == "" {
		panic(jr.ctx.vm.ToValue("PromptUserChannelThreadForReply: user must not be empty"))
	}

	// Argument #2: channel (string)
	channel := jr.requireStringArg(methodName, call, 2)
	if channel == "" {
		panic(jr.ctx.vm.ToValue("PromptUserChannelThreadForReply: channel must not be empty"))
	}

	// Argument #3: thread (string)
	thread := jr.requireStringArg(methodName, call, 3)
	// We allow empty thread, so no immediate panic here.

	// Argument #4: prompt (string), optional
	var prompt string
	if len(call.Arguments) > 4 {
		rawPrompt := call.Arguments[4].Export()
		if str, ok := rawPrompt.(string); ok {
			prompt = str
		} else {
			panic(jr.ctx.vm.ToValue(fmt.Sprintf(
				"%s: argument #5 (prompt) must be a string if provided",
				methodName,
			)))
		}
	}

	reply, ret := jr.r.PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt)

	res := jr.ctx.vm.NewObject()
	res.Set("reply", reply)
	res.Set("retVal", int(ret))
	return res
}
