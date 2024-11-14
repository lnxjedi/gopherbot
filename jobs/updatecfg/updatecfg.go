package main

import "github.com/lnxjedi/gopherbot/robot"

func JobHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	r.Log(robot.Warn, "Deprecated updatecfg job ran, adding job 'go-update'")
	r.AddJob("go-update")
	return robot.Normal
}
