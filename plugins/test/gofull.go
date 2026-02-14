package main

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
)

var defaultConfig = []byte(`
---
Help:
- Keywords: [ "say", "ask" ]
  Helptext:
  - "(bot), say everything - full test of Say*/Reply*/Send* methods"
- Keywords: [ "config" ]
  Helptext:
  - "(bot), go-config - exercise GetTaskConfig + RandomString"
- Keywords: [ "subscribe" ]
  Helptext:
  - "(bot), go-subscribe - exercise Subscribe/Unsubscribe"
CommandMatchers:
- Regex: (?i:say everything)
  Command: sendmsg
- Regex: (?i:go-config)
  Command: configtest
- Regex: (?i:go-subscribe)
  Command: subscribe
AllowedHiddenCommands:
- sendmsg
Config:
  Openings:
  - "Not completely random 1"
  - "Not completely random 2"
`)

type goFullConfig struct {
	Openings []string
}

func Configure() *[]byte {
	return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	switch command {
	case "init":
		return robot.Normal
	case "sendmsg":
		msg := r.GetMessage()
		if msg == nil {
			return robot.Fail
		}
		r.Say("Regular Say")
		r.SayThread("SayThread, yeah")
		r.Reply("Regular Reply")
		r.ReplyThread("Reply in thread, yo")
		r.SendChannelMessage(msg.Channel, "Sending to the channel: %s", msg.Channel)
		r.SendUserMessage(msg.User, "Sending this message to user: %s", msg.User)
		r.SendUserChannelMessage(msg.User, msg.Channel, "Sending to user '%s' in channel: %s", msg.User, msg.Channel)
		r.SendChannelThreadMessage(msg.Channel, "0xDEADBEEF", "Sending to channel '%s' in thread: 0xDEADBEEF", msg.Channel)
		r.SendUserChannelThreadMessage(msg.User, msg.Channel, "0xDEADBEEF", "Sending to user '%s' in channel '%s' in thread: 0xDEADBEEF", msg.User, msg.Channel)
		return robot.Normal
	case "configtest":
		var cfg goFullConfig
		if ret := r.GetTaskConfig(&cfg); ret != robot.Ok {
			r.Say("No config available")
			return robot.Fail
		}
		r.Say(r.RandomString(cfg.Openings))
		return robot.Normal
	case "subscribe":
		sub := r.Subscribe()
		unsub := r.Unsubscribe()
		r.Say(fmt.Sprintf("SUBSCRIBE FLOW: %t/%t", sub, unsub))
		return robot.Normal
	default:
		return robot.Fail
	}
}
