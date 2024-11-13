package main

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
)

// PluginHandler is the entry point for the plugin.
func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {
	// Test a few Robot methods
	if r.CheckAdmin() {
		r.Say("You are an admin!")
	} else {
		r.Say("You are not an admin.")
	}

	attrRet := r.GetBotAttribute("name")
	if attrRet.RetVal == robot.Ok {
		r.Say(fmt.Sprintf("Bot Name: %s", attrRet.Attribute))
	} else {
		r.Say("Failed to get bot name.")
	}

	return robot.Normal
}
