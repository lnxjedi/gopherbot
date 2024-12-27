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
		"PromptUserChannelThreadForReply": lctx.botPromptUserChannelThreadForReply,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
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
	user := L.CheckString(3)
	channel := L.CheckString(4)
	threadID := L.CheckString(5)
	prompt := L.CheckString(6)

	lr, ok := ud.Value.(*luaRobot)
	if !ok || lr == nil || lr.r == nil {
		lctx.logBotErr("PromptUserChannelThreadForReply")
		return pushPromptFail(L)
	}

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
