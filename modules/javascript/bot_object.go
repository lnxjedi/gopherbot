// bot_object.go
package javascript

import (
	"github.com/dop251/goja"
	"github.com/lnxjedi/gopherbot/robot"
)

type jsBot struct {
	r   robot.Robot
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
	botObj.Set("SendUserChannelThreadMessage", jr.botSendUserChannelThreadMessage)
	botObj.Set("Say", jr.botSay)
	botObj.Set("SayThread", jr.botSayThread)
	botObj.Set("Reply", jr.botReply)
	botObj.Set("ReplyThread", jr.botReplyThread)
	botObj.Set("Fixed", jr.botFixed)
	botObj.Set("Direct", jr.botDirect)
	botObj.Set("Threaded", jr.botThreaded)
	botObj.Set("MessageFormat", jr.botMessageFormat)

	return botObj
}
