// bot_object.go
package javascript

import (
	"github.com/dop251/goja"
)

type jsBot struct {
	r   BotAPI
	ctx *jsContext
}

// registerBotObject - exposes a global "bot" in JS with all the properties
// we need to implement a proper bot in JavaScript, and all the methods needed
// for the Gopherbot robot API.
func (jr *jsBot) createBotObject() *goja.Object {
	botObj := jr.ctx.vm.NewObject()

	// Set string fields directly from ctx.bot
	keys := []string{"user", "user_id", "channel", "channel_id", "thread_id", "message_id", "protocol", "brain"}
	for _, key := range keys {
		if value, ok := jr.ctx.bot[key]; ok {
			botObj.Set(key, value)
		}
	}

	// Set threaded_message boolean based on ctx.bot["GOPHER_THREADED_MESSAGE"]
	if jr.ctx.bot["threaded_message"] == "true" {
		botObj.Set("threaded_message", true)
	} else {
		botObj.Set("threaded_message", false)
	}
	// This is true when loading config, where no robots are created or
	// used.
	if jr.r == nil {
		return botObj
	}

	botObj.Set("SendChannelMessage", jr.botSendChannelMessage)
	botObj.Set("SendChannelThreadMessage", jr.botSendChannelThreadMessage)
	botObj.Set("SendUserMessage", jr.botSendUserMessage)
	botObj.Set("SendUserChannelMessage", jr.botSendUserChannelMessage)
	botObj.Set("SendProtocolUserChannelMessage", jr.botSendProtocolUserChannelMessage)
	botObj.Set("SendUserChannelThreadMessage", jr.botSendUserChannelThreadMessage)
	botObj.Set("Say", jr.botSay)
	botObj.Set("SayThread", jr.botSayThread)
	botObj.Set("Reply", jr.botReply)
	botObj.Set("ReplyThread", jr.botReplyThread)
	botObj.Set("Fixed", jr.botFixed)
	botObj.Set("Direct", jr.botDirect)
	botObj.Set("Threaded", jr.botThreaded)
	botObj.Set("MessageFormat", jr.botMessageFormat)
	botObj.Set("CheckoutDatum", jr.botCheckoutDatum)
	botObj.Set("UpdateDatum", jr.botUpdateDatum)
	botObj.Set("CheckinDatum", jr.botCheckinDatum)
	botObj.Set("GetBotAttribute", jr.botGetBotAttribute)
	botObj.Set("GetUserAttribute", jr.botGetUserAttribute)
	botObj.Set("GetSenderAttribute", jr.botGetSenderAttribute)
	botObj.Set("GetTaskConfig", jr.botGetTaskConfig)
	botObj.Set("RandomInt", jr.botRandomInt)
	botObj.Set("RandomString", jr.botRandomString)
	botObj.Set("Pause", jr.botPause)
	botObj.Set("CheckAdmin", jr.botCheckAdmin)
	botObj.Set("Elevate", jr.botElevate)
	botObj.Set("Log", jr.botLog)
	botObj.Set("Remember", jr.botRemember)
	botObj.Set("RememberThread", jr.botRememberThread)
	botObj.Set("RememberContext", jr.botRememberContext)
	botObj.Set("RememberContextThread", jr.botRememberContextThread)
	botObj.Set("Recall", jr.botRecall)
	botObj.Set("GetParameter", jr.botGetParameter)
	botObj.Set("SetParameter", jr.botSetParameter)
	botObj.Set("Exclusive", jr.botExclusive)
	botObj.Set("SpawnJob", jr.botSpawnJob)
	botObj.Set("AddTask", jr.botAddTask)
	botObj.Set("FinalTask", jr.botFinalTask)
	botObj.Set("FailTask", jr.botFailTask)
	botObj.Set("AddJob", jr.botAddJob)
	botObj.Set("AddCommand", jr.botAddCommand)
	botObj.Set("FinalCommand", jr.botFinalCommand)
	botObj.Set("FailCommand", jr.botFailCommand)
	botObj.Set("PromptForReply", jr.botPromptForReply)
	botObj.Set("PromptThreadForReply", jr.botPromptThreadForReply)
	botObj.Set("PromptUserForReply", jr.botPromptUserForReply)
	botObj.Set("PromptUserChannelForReply", jr.botPromptUserChannelForReply)
	botObj.Set("PromptUserChannelThreadForReply", jr.botPromptUserChannelThreadForReply)

	return botObj
}
