// send_messages.go
package javascript

import (
	"github.com/dop251/goja"
)

// botSendChannelMessage(bot:SendChannelMessage("channel", "message"))
// The engine handles empty messages (returns robot.Fail / logs a warning).
func (jr *jsBot) botSendChannelMessage(call goja.FunctionCall) goja.Value {
	const methodName = "SendChannelMessage"

	channel := jr.requireStringArg(methodName, call, 0)
	msg := jr.requireStringArg(methodName, call, 1)

	// Hard fail on empty channel
	if channel == "" {
		panic(jr.ctx.vm.ToValue("SendChannelMessage: channel name must not be empty"))
	}

	ret := jr.r.SendChannelMessage(channel, msg)
	return jr.ctx.vm.ToValue(int(ret))
}

// botSendChannelThreadMessage(bot:SendChannelThreadMessage("channel", "thread", "message"))
// The engine handles empty messages. The "thread" can be empty => top-level.
func (jr *jsBot) botSendChannelThreadMessage(call goja.FunctionCall) goja.Value {
	const methodName = "SendChannelThreadMessage"

	channel := jr.requireStringArg(methodName, call, 0)
	thread := jr.requireStringArg(methodName, call, 1)
	msg := jr.requireStringArg(methodName, call, 2)

	if channel == "" {
		panic(jr.ctx.vm.ToValue("SendChannelThreadMessage: channel must not be empty"))
	}

	ret := jr.r.SendChannelThreadMessage(channel, thread, msg)
	return jr.ctx.vm.ToValue(int(ret))
}

// botSendUserMessage(bot:SendUserMessage("some.user", "message"))
// The engine handles empty messages. If user is empty, we fail fast.
func (jr *jsBot) botSendUserMessage(call goja.FunctionCall) goja.Value {
	const methodName = "SendUserMessage"

	user := jr.requireStringArg(methodName, call, 0)
	msg := jr.requireStringArg(methodName, call, 1)

	if user == "" {
		panic(jr.ctx.vm.ToValue("SendUserMessage: user argument must not be empty"))
	}

	ret := jr.r.SendUserMessage(user, msg)
	return jr.ctx.vm.ToValue(int(ret))
}

// botSendUserChannelMessage(bot:SendUserChannelMessage("some.user", "some-channel", "message"))
// The engine handles empty messages. Must not have empty user or channel.
func (jr *jsBot) botSendUserChannelMessage(call goja.FunctionCall) goja.Value {
	const methodName = "SendUserChannelMessage"

	user := jr.requireStringArg(methodName, call, 0)
	channel := jr.requireStringArg(methodName, call, 1)
	msg := jr.requireStringArg(methodName, call, 2)

	if user == "" {
		panic(jr.ctx.vm.ToValue("SendUserChannelMessage: user must not be empty"))
	}
	if channel == "" {
		panic(jr.ctx.vm.ToValue("SendUserChannelMessage: channel must not be empty"))
	}

	ret := jr.r.SendUserChannelMessage(user, channel, msg)
	return jr.ctx.vm.ToValue(int(ret))
}

// botSendUserChannelThreadMessage(bot:SendUserChannelThreadMessage("some.user", "some-channel", "thread", "message"))
// The engine handles empty messages, logs warnings, etc.
// Must not have empty user or channel; thread can be empty => top-level.
func (jr *jsBot) botSendUserChannelThreadMessage(call goja.FunctionCall) goja.Value {
	const methodName = "SendUserChannelThreadMessage"

	user := jr.requireStringArg(methodName, call, 0)
	channel := jr.requireStringArg(methodName, call, 1)
	thread := jr.requireStringArg(methodName, call, 2)
	msg := jr.requireStringArg(methodName, call, 3)

	if user == "" {
		panic(jr.ctx.vm.ToValue("SendUserChannelThreadMessage: user must not be empty"))
	}
	if channel == "" {
		panic(jr.ctx.vm.ToValue("SendUserChannelThreadMessage: channel must not be empty"))
	}

	ret := jr.r.SendUserChannelThreadMessage(user, channel, thread, msg)
	return jr.ctx.vm.ToValue(int(ret))
}

// botSay(bot:Say("some text"))
// The engine handles empty messages => returns robot.Fail or logs a warning.
func (jr *jsBot) botSay(call goja.FunctionCall) goja.Value {
	const methodName = "Say"

	msg := jr.requireStringArg(methodName, call, 0)
	ret := jr.r.Say(msg)
	return jr.ctx.vm.ToValue(int(ret))
}

// botSayThread(bot:SayThread("some text"))
// The engine handles empty messages => returns fail / logs a warning.
func (jr *jsBot) botSayThread(call goja.FunctionCall) goja.Value {
	const methodName = "SayThread"

	msg := jr.requireStringArg(methodName, call, 0)
	ret := jr.r.SayThread(msg)
	return jr.ctx.vm.ToValue(int(ret))
}

// botReply(bot:Reply("some text"))
// The engine handles empty messages => logs warning / returns fail.
func (jr *jsBot) botReply(call goja.FunctionCall) goja.Value {
	const methodName = "Reply"

	msg := jr.requireStringArg(methodName, call, 0)
	ret := jr.r.Reply(msg)
	return jr.ctx.vm.ToValue(int(ret))
}

// botReplyThread(bot:ReplyThread("some text"))
// The engine handles empty messages => logs warning / returns fail.
func (jr *jsBot) botReplyThread(call goja.FunctionCall) goja.Value {
	const methodName = "ReplyThread"

	msg := jr.requireStringArg(methodName, call, 0)
	ret := jr.r.ReplyThread(msg)
	return jr.ctx.vm.ToValue(int(ret))
}
