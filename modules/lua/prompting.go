package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// RegisterPromptingMethod adds the prompting method to the "bot" metatable:
//	bot:PromptUserChannelThreadForReply(regexID, prompt)
func (lctx luaContext) RegisterPromptingMethods(L *glua.LState) {
	methods := map[string]glua.LGFunction{
		"PromptUserChannelThreadForReply": lctx.botPromptUserChannelThreadForReply,
	}

	mt := registerBotMetatableIfNeeded(L)
	L.SetFuncs(mt, methods)
}

// botPromptUserChannelThreadForReply:
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
		L.Push(glua.LString(""))
		L.Push(glua.LNumber(robot.FailedMessageSend))
		return 2
	}

	reply, ret := lr.r.PromptUserChannelThreadForReply(regexID, user, channel, threadID, prompt)
	L.Push(glua.LString(reply))
	L.Push(glua.LNumber(ret))
	return 2
}
