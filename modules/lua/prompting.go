package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterPromptingMethods adds the prompt methods to the "bot" metatable:
//
//	bot:PromptForReply(regexID, prompt)
//	bot:PromptThreadForReply(regexID, prompt)
//	bot:PromptUserForReply(regexID, prompt)
//	bot:PromptUserChannelForReply(regexID, prompt)
//	bot:PromptUserChannelThreadForReply(regexID, prompt)
//
// Each method uses the bot's own user/channel/thread fields,
// just like in your Ruby examples.
func (lctx luaContext) RegisterPromptingMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"PromptForReply":                  lctx.botPromptForReply,
		"PromptThreadForReply":            lctx.botPromptThreadForReply,
		"PromptUserForReply":              lctx.botPromptUserForReply,
		"PromptUserChannelForReply":       lctx.botPromptUserChannelForReply,
		"PromptUserChannelThreadForReply": lctx.botPromptUserChannelThreadForReply,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// -------------------------------------------------------------------
// 1. Basic prompting methods that mirror your Ruby code
// -------------------------------------------------------------------

// botPromptForReply:
// Ruby-ish:
//
//	def PromptForReply(regex_id, prompt)
//	    thread = @threaded_message ? @thread_id : ""
//	    PromptUserChannelThreadForReply(regex_id, @user, @channel, thread, prompt)
//	end
func (lctx luaContext) botPromptForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("PromptForReply")
		return pushPromptFail(L)
	}

	user, _ := lr.fields["user"].(string)
	channel, _ := lr.fields["channel"].(string)
	threaded, _ := lr.fields["threaded_message"].(bool)
	threadID, _ := lr.fields["thread_id"].(string)

	// If the bot is threaded, use thread_id, else empty string
	thread := ""
	if threaded {
		thread = threadID
	}

	reply, ret := lr.r.PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// botPromptThreadForReply:
// Ruby-ish:
//
//	def PromptThreadForReply(regex_id, prompt)
//	    PromptUserChannelThreadForReply(regex_id, @user, @channel, @thread_id, prompt)
//	end
func (lctx luaContext) botPromptThreadForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("PromptThreadForReply")
		return pushPromptFail(L)
	}

	user, _ := lr.fields["user"].(string)
	channel, _ := lr.fields["channel"].(string)
	threadID, _ := lr.fields["thread_id"].(string)

	reply, ret := lr.r.PromptUserChannelThreadForReply(regexID, user, channel, threadID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// botPromptUserForReply:
// Ruby-ish:
//
//	def PromptUserForReply(regex_id, prompt)
//	    PromptUserChannelThreadForReply(regex_id, @user, "", "", prompt)
//	end
func (lctx luaContext) botPromptUserForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("PromptUserForReply")
		return pushPromptFail(L)
	}

	user, _ := lr.fields["user"].(string)

	reply, ret := lr.r.PromptUserChannelThreadForReply(regexID, user, "", "", prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// botPromptUserChannelForReply:
// Ruby-ish (if we wanted a separate method ignoring thread):
//
//	def PromptUserChannelForReply(regex_id, prompt)
//	    PromptUserChannelThreadForReply(regex_id, @user, @channel, "", prompt)
//	end
func (lctx luaContext) botPromptUserChannelForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("PromptUserChannelForReply")
		return pushPromptFail(L)
	}

	user, _ := lr.fields["user"].(string)
	channel, _ := lr.fields["channel"].(string)

	reply, ret := lr.r.PromptUserChannelThreadForReply(regexID, user, channel, "", prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// botPromptUserChannelThreadForReply:
// Ruby-ish:
//
//	def PromptUserChannelThreadForReply(regex_id, prompt)
//	    PromptUserChannelThreadForReply(regex_id, @user, @channel, @thread_id, prompt)
//	end
func (lctx luaContext) botPromptUserChannelThreadForReply(L *glua.LState) int {
	ud := L.CheckUserData(1)
	regexID := L.CheckString(2)
	prompt := L.CheckString(3)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("PromptUserChannelThreadForReply")
		return pushPromptFail(L)
	}

	user, _ := lr.fields["user"].(string)
	channel, _ := lr.fields["channel"].(string)
	threadID, _ := lr.fields["thread_id"].(string)

	reply, ret := lr.r.PromptUserChannelThreadForReply(regexID, user, channel, threadID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}

// pushPromptFail is a helper to push an empty string and FailedMessageSend
func pushPromptFail(L *glua.LState) int {
	L.Push(glua.LString(""))
	L.Push(glua.LNumber(robot.FailedMessageSend))
	return 2
}
