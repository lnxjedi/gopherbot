package main

import (
	"github.com/lnxjedi/gopherbot/robot"
	"gopherbot.internal/lib/newrobotflow"
)

func Configure() *[]byte {
	return &newrobotflow.StartPluginConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	switch command {
	case "init":
		return
	case newrobotflow.CommandStart, newrobotflow.CommandCancel:
		newrobotflow.HandleStartCommand(r, command)
	}
	return
}
