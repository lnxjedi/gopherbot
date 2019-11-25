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
func (c *botContext) checkJobMatchersAndRun() (messageMatched bool) {
	r := c.makeRobot()
	// un-needed, but more clear
	messageMatched = false
	runTasks := []interface{}{}
	robots := []*botContext{}
	taskArgs := [][]string{}
	var triggerArgs []string

	// First, check triggers
	for _, t := range c.tasks.t {
		task, _, job := getTask(t)
		if job == nil {
			continue
		}
		if task.Disabled {
			msg := fmt.Sprintf("Skipping disabled job '%s', reason: %s", task.name, task.reason)
			Log(robot.Trace, msg)
			c.debugT(t, msg, false)
			continue
		}
		Log(robot.Trace, "Checking triggers for job '%s'", task.name)
		triggers := job.Triggers
		c.debugT(t, fmt.Sprintf("Checking %d JobTriggers against message: '%s' from user '%s' in channel '%s'", len(triggers), c.msg, c.User, c.Channel), false)
		for _, trigger := range triggers {
			Log(robot.Trace, "Checking '%s' against user '%s', channel '%s', regex: '%s'", c.msg, trigger.User, trigger.Channel, trigger.Regex)
			if c.User != trigger.User {
				c.debugT(t, fmt.Sprintf("User '%s' doesn't match trigger user '%s'", c.User, trigger.User), false)
				continue
			}
			if c.Channel != trigger.Channel {
				c.debugT(t, fmt.Sprintf("Channel '%s' doesn't match trigger", c.Channel), false)
				continue
			}
			matches := trigger.re.FindAllStringSubmatch(c.msg, -1)
			matched := false
			if matches != nil {
				c.debugT(t, fmt.Sprintf("Matched trigger regex '%s'", trigger.Regex), false)
				Log(robot.Trace, "Message '%s' matches trigger for job '%s'", c.msg, task.name)
				matched = true
				triggerArgs = matches[0][1:]
			} else {
				c.debugT(t, fmt.Sprintf("Not matched: %s", trigger.Regex), false)
			}
			if matched {
				messageMatched = true
				newbot := c.clone()
				newbot.automaticTask = true
				robots = append(robots, newbot)
				runTasks = append(runTasks, t)
				taskArgs = append(taskArgs, triggerArgs)
			}
		} // end of triggerer checking
	} // end of job trigger checking
	if messageMatched {
		botCfg.RLock()
		if botCfg.shuttingDown {
			r.Say("Ignoring triggered job(s): shutting down")
			botCfg.RUnlock()
			return
		}
		botCfg.RUnlock()
		if len(robots) > 0 {
			for i, robot := range robots {
				go robot.startPipeline(nil, runTasks[i], jobTrigger, "run", taskArgs[i]...)
			}
		}
		return
	}
	// Check for built-in run job
	if c.isCommand {
		var jobName string
		cmsg := spaceRe.ReplaceAllString(c.msg, " ")
		matches := runJobRe.FindAllStringSubmatch(cmsg, -1)
		if matches != nil {
			jobName = matches[0][1]
			messageMatched = true
			c.messageHeard()
		} else {
			return
		}
		t := c.jobAvailable(jobName)
		if t != nil {
			c.currentTask = t
			c.registerActive(nil)
			r := c.makeRobot()
			task, _, job := getTask(t)
			if task.Disabled {
				r.Say("Job '%s' is disabled: %s", jobName, task.reason)
				c.deregister()
				return
			}
			if !c.jobSecurityCheck(t, "run") {
				c.deregister()
				return
			}
			var args []string
			// remember which job we're talking about
			ctx := memoryContext{"context:task", c.User, c.Channel}
			s := shortTermMemory{jobName, time.Now()}
			shortTermMemories.Lock()
			shortTermMemories.m[ctx] = s
			shortTermMemories.Unlock()
			if len(matches[0][2]) > 0 { // arguments supplied with `run job foo bar baz`, check match to arguments
				args = strings.Split(matches[0][2], " ")
				if len(args) != len(job.Arguments) {
					r.Say("Wrong number of arguments for job '%s', %d configured but %d given", jobName, len(job.Arguments), len(args))
					c.deregister()
					return
				}
				for i, arg := range args {
					if !job.Arguments[i].re.MatchString(arg) {
						r.Say("'%s' doesn't match the pattern for argument '%s'", arg, job.Arguments[i].Label)
						c.deregister()
						return
					}
				}
			} else {
				if len(job.Arguments) > 0 {
					args = make([]string, len(job.Arguments))
					c.currentTask = t
					c.pipeName = task.name
					c.pipeDesc = task.Description
					r = c.makeRobot()
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
									c.deregister()
									return
								}
								args[i] = arg
								break
							}
						}
						if t == 3 {
							r.Say("(giving up)")
							c.deregister()
							return
						}
					}
				}
			}
			c.deregister()
			c.verbose = true
			c.startPipeline(nil, t, jobCmd, "run", args...)
		} // jobAvailable sends a message if it's not
	}
	return
}
