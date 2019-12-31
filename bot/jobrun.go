package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const runJobRegex = `run +job +(` + identifierRegex + `)(?: (.*))?`

var runJobRe = regexp.MustCompile(`(?i:^\s*` + runJobRegex + `\s*$)`)

// checkJobMatchersAndRun handles triggers, 'run job <foo>'
func (w *worker) checkJobMatchersAndRun() (messageMatched bool) {
	// un-needed, but more clear
	messageMatched = false
	runTasks := []interface{}{}
	robots := []*worker{}
	taskArgs := [][]string{}
	var triggerArgs []string

	// First, check triggers
	for _, t := range w.tasks.t[1:] {
		task, _, job := getTask(t)
		if job == nil {
			continue
		}
		if task.Disabled {
			msg := fmt.Sprintf("Skipping disabled job '%s', reason: %s", task.name, task.reason)
			Log(robot.Trace, msg)
			debugT(t, msg, false)
			continue
		}
		Log(robot.Trace, "Checking triggers for job '%s'", task.name)
		triggers := job.Triggers
		debugT(t, fmt.Sprintf("Checking %d JobTriggers against message: '%s' from user '%s' in channel '%s'", len(triggers), w.msg, w.User, w.Channel), false)
		for _, trigger := range triggers {
			Log(robot.Trace, "Checking '%s' against user '%s', channel '%s', regex: '%s'", w.msg, trigger.User, trigger.Channel, trigger.Regex)
			if w.User != trigger.User {
				debugT(t, fmt.Sprintf("User '%s' doesn't match trigger user '%s'", w.User, trigger.User), false)
				continue
			}
			if w.Channel != trigger.Channel {
				debugT(t, fmt.Sprintf("Channel '%s' doesn't match trigger", w.Channel), false)
				continue
			}
			matches := trigger.re.FindAllStringSubmatch(w.msg, -1)
			matched := false
			if matches != nil {
				debugT(t, fmt.Sprintf("Matched trigger regex '%s'", trigger.Regex), false)
				Log(robot.Trace, "Message '%s' matches trigger for job '%s'", w.msg, task.name)
				matched = true
				triggerArgs = matches[0][1:]
			} else {
				debugT(t, fmt.Sprintf("Not matched: %s", trigger.Regex), false)
			}
			if matched {
				messageMatched = true
				newbot := w.clone()
				newbot.automaticTask = true
				robots = append(robots, newbot)
				runTasks = append(runTasks, t)
				taskArgs = append(taskArgs, triggerArgs)
			}
		} // end of triggerer checking
	} // end of job trigger checking
	if messageMatched {
		state.RLock()
		if state.shuttingDown {
			w.Say("Ignoring triggered job(s): shutting down")
			state.RUnlock()
			return
		}
		state.RUnlock()
		if len(robots) > 0 {
			for i, robot := range robots {
				go robot.startPipeline(nil, runTasks[i], jobTrigger, "run", taskArgs[i]...)
			}
		}
		return
	}
	// Check for built-in run job
	if w.isCommand {
		var jobName string
		cmsg := spaceRe.ReplaceAllString(w.msg, " ")
		matches := runJobRe.FindAllStringSubmatch(cmsg, -1)
		if matches != nil {
			jobName = matches[0][1]
			messageMatched = true
			w.messageHeard()
		} else {
			return
		}
		t := w.jobAvailable(jobName)
		if t != nil {
			c := &pipeContext{
				environment: make(map[string]string),
			}
			w.pipeContext = c
			c.currentTask = t
			// We need an active worker in case we need to call possibly
			// external authorizer or elevator.
			w.registerActive(nil)
			// REMOVE ME r := w.makeRobot()
			task, _, job := getTask(t)
			if task.Disabled {
				w.Say("Job '%s' is disabled: %s", jobName, task.reason)
				w.deregister()
				return
			}
			if !w.jobSecurityCheck(t, "run") {
				w.deregister()
				return
			}
			var args []string
			// remember which job we're talking about
			ctx := memoryContext{"context:task", w.User, w.Channel}
			s := shortTermMemory{jobName, time.Now()}
			shortTermMemories.Lock()
			shortTermMemories.m[ctx] = s
			shortTermMemories.Unlock()
			if len(matches[0][2]) > 0 { // arguments supplied with `run job foo bar baz`, check match to arguments
				args = strings.Split(matches[0][2], " ")
				if len(args) != len(job.Arguments) {
					w.Say("Wrong number of arguments for job '%s', %d configured but %d given", jobName, len(job.Arguments), len(args))
					w.deregister()
					return
				}
				for i, arg := range args {
					if !job.Arguments[i].re.MatchString(arg) {
						w.Say("'%s' doesn't match the pattern for argument '%s'", arg, job.Arguments[i].Label)
						w.deregister()
						return
					}
				}
			} else {
				if len(job.Arguments) > 0 {
					args = make([]string, len(job.Arguments))
					c.currentTask = t
					c.pipeName = task.name
					c.pipeDesc = task.Description
					r := w.makeRobot()
					for i, argspec := range job.Arguments {
						var t int
						for t = 1; t < 3; t++ {
							arg, ret := r.PromptForReply(argspec.Label, fmt.Sprintf("What's the value for '%s'?", argspec.Label))
							if ret == robot.ReplyNotMatched {
								r.Say("That doesn't match the pattern for argument '%s'", argspec.Label)
							} else {
								if ret != robot.Ok {
									r.Log(robot.Warn, "failed getting arguments running job '%s': %s", jobName, ret)
									r.Say("(not running job '%s')", jobName)
									w.deregister()
									return
								}
								args[i] = arg
								break
							}
						}
						if t == 3 {
							r.Say("(giving up)")
							w.deregister()
							return
						}
					}
				}
			}
			w.deregister()
			c.verbose = true
			w.startPipeline(nil, t, jobCmd, "run", args...)
		} // jobAvailable sends a message if it's not
	}
	return
}
