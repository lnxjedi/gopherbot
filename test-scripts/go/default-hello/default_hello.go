package main

import "github.com/lnxjedi/gopherbot/robot"

var defaultConfig = []byte(`---
Commands:
  - Regex: (?i:go default hello)
    Command: hello
`)

func Configure() *[]byte {
	return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {
	switch command {
	case "_init":
		return robot.Normal
	case "hello":
		msg := r.GetMessage()
		channel := "unknown"
		if msg != nil && msg.Channel != "" {
			channel = msg.Channel
		}
		botName := r.GetBotAttribute("fullName").Attribute
		sender := r.GetSenderAttribute("fullName").Attribute
		r.Say("Go says hello from %s to %s in #%s.", botName, sender, channel)
		r.Reply("This script only needs the installed default fixture.")
		return robot.Normal
	default:
		r.Log(robot.Error, "unknown Go default command: %s", command)
		return robot.Fail
	}
}
