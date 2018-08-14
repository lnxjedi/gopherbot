package bot

import (
	"fmt"
	"regexp"
	"time"
)

const runJobRegex = `run +job +(` + identifierRegex + `)`

var runJobRe = regexp.MustCompile(`(?i:^\s*` + runJobRegex + `\s*$)`)

// checkJobMatchersAndRun handles triggers, 'run job <foo>', 'history <foo>'
func (bot *botContext) checkJobMatchersAndRun() (messageMatched bool) {
	r := bot.makeRobot()
	// un-needed, but more clear
	messageMatched = false
	runTasks := []interface{}{}
	robots := []*botContext{}
	taskArgs := [][]string{}
	var triggerArgs []string

	// First, check triggers
	for _, t := range bot.tasks.t {
		task, _, job := getTask(t)
		if job == nil {
			continue
		}
		if task.Disabled {
			msg := fmt.Sprintf("Skipping disabled job '%s', reason: %s", task.name, task.reason)
			Log(Trace, msg)
			bot.debugT(t, msg, false)
			continue
		}
		Log(Trace, fmt.Sprintf("Checking triggers for job '%s'", task.name))
		triggers := job.Triggers
		bot.debugT(t, fmt.Sprintf("Checking %d JobTriggers against message: '%s' from user '%s' in channel '%s'", len(triggers), bot.msg, bot.User, bot.Channel), false)
		for _, trigger := range triggers {
			Log(Trace, fmt.Sprintf("Checking '%s' against user '%s', channel '%s', regex: '%s'", bot.msg, trigger.User, trigger.Channel, trigger.Regex))
			if bot.User != trigger.User {
				bot.debugT(t, fmt.Sprintf("User '%s' doesn't match trigger user '%s'", bot.User, trigger.User), false)
				continue
			}
			if bot.Channel != trigger.Channel {
				bot.debugT(t, fmt.Sprintf("Channel '%s' doesn't match trigger", bot.Channel), false)
				continue
			}
			matches := trigger.re.FindAllStringSubmatch(bot.msg, -1)
			matched := false
			if matches != nil {
				bot.debugT(t, fmt.Sprintf("Matched trigger regex '%s'", trigger.Regex), false)
				Log(Trace, fmt.Sprintf("Message '%s' matches trigger for job '%s'", bot.msg, task.name))
				matched = true
				triggerArgs = matches[0][1:]
			} else {
				bot.debugT(t, fmt.Sprintf("Not matched: %s", trigger.Regex), false)
			}
			if matched {
				messageMatched = true
				newbot := bot.clone()
				newbot.automaticTask = true
				robots = append(robots, newbot)
				runTasks = append(runTasks, t)
				taskArgs = append(taskArgs, triggerArgs)
			}
		} // end of triggerer checking
	} // end of job trigger checking
	if messageMatched {
		robot.RLock()
		if robot.shuttingDown {
			r.Say("Ignoring triggered job(s): shutting down")
			robot.RUnlock()
			return
		} else if robot.paused {
			r.Say("Ignoring triggered job(s): paused")
			robot.RUnlock()
			return
		}
		robot.RUnlock()
		if len(robots) > 0 {
			for i, robot := range robots {
				go robot.startPipeline(nil, runTasks[i], jobTrigger, "run", taskArgs[i]...)
			}
		}
		return
	}
	// Check for built-in run job
	if bot.isCommand {
		var jobName string
		cmsg := spaceRe.ReplaceAllString(bot.msg, " ")
		matches := runJobRe.FindAllStringSubmatch(cmsg, -1)
		if matches != nil {
			jobName = matches[0][1]
			messageMatched = true
			r.messageHeard()
		} else {
			return
		}
		t := bot.jobAvailable(jobName, false)
		if t != nil {
			// remember which job we're talking about
			ctx := memoryContext{"context:task", bot.User, bot.Channel}
			s := shortTermMemory{jobName, time.Now()}
			shortTermMemories.Lock()
			shortTermMemories.m[ctx] = s
			shortTermMemories.Unlock()

			bot.startPipeline(nil, t, jobCmd, "run")
		} // jobAvailable sends a message if it's not
	}
	return
}
