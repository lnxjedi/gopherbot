package main

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
)

var defaultConfig = []byte(`---
Commands:
  - Regex: (?i:go local demo)(?: (.*))?
    Command: demo
  - Regex: (?i:go local prompt)
    Command: prompt
  - Regex: (?i:go local memory)
    Command: memory
  - Regex: (?i:go local config)
    Command: config
`)

type demoConfig struct {
	Openings      []string
	FavoritePlace string
	Retries       int
}

func Configure() *[]byte {
	return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {
	switch command {
	case "_init":
		return robot.Normal
	case "demo":
		sender := r.GetSenderAttribute("fullName")
		color := r.GetParameter("CAT_COLOR")
		r.Say("Go demo for %s; cat color=%s.", sender.Attribute, color)
		if len(args) > 0 {
			r.Reply("First capture arg was %s", args[0])
		}
		r.Fixed().SendChannelMessage(r.GetMessage().Channel, "fixed-width channel message from Go")
		return robot.Normal
	case "prompt":
		name, ret := r.PromptForReply("SimpleString", "Name of cat?")
		if ret != robot.Ok {
			r.Say("Prompt failed: %s", ret)
			return robot.Fail
		}
		r.Say("Go heard cat name: %s", name)
		return robot.Normal
	case "memory":
		r.Remember("last_cat", "Garfield", false)
		r.Remember("team_cat", "Pixel", true)
		r.Say("Go memory local=%s shared=%s", r.Recall("last_cat", false), r.Recall("team_cat", true))
		var profile map[string]interface{}
		token, exists, ret := r.CheckoutDatum("cat_profile", &profile, true)
		if ret == robot.Ok {
			if !exists || profile == nil {
				profile = map[string]interface{}{}
			}
			profile["name"] = "Garfield"
			profile["snack"] = "lasagna"
			_ = r.UpdateDatum("cat_profile", token, profile)
		}
		return robot.Normal
	case "config":
		var cfg demoConfig
		if ret := r.GetTaskConfig(&cfg); ret != robot.Ok {
			r.Say("Go config unavailable: %s", ret)
			return robot.Fail
		}
		opening := "<missing>"
		if len(cfg.Openings) > 0 {
			opening = cfg.Openings[0]
		}
		r.Say("Go config opening: %s", opening)
		r.SetParameter("GO_LOCAL_DEMO", "ok")
		r.AddTask("next-task", "from-go")
		return robot.Normal
	default:
		r.Log(robot.Error, fmt.Sprintf("unknown Go local command: %s", command))
		return robot.Fail
	}
}
