package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// (?i:^\s*run[- ]job ([A-Za-z][\w-]*)(?: (.*))?\s*$)
const runJobRegex = `run[- ]job (` + identifierRegex + `)(?: (.*))?`

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
			continue
		}
		Log(robot.Trace, "Checking triggers for job '%s'", task.name)
		triggers := job.Triggers
		for _, trigger := range triggers {
			Log(robot.Trace, "Checking '%s' against user '%s', channel '%s', regex: '%s'", w.msg, trigger.User, trigger.Channel, trigger.Regex)
			if w.User != trigger.User {
				continue
			}
			if w.Channel != trigger.Channel {
				continue
			}
			matches := trigger.re.FindAllStringSubmatch(w.msg, -1)
			matched := false
			if matches != nil {
				Log(robot.Trace, "Message '%s' matches trigger for job '%s'", w.msg, task.name)
				matched = true
				triggerArgs = matches[0][1:]
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
	// Messages from the robot itself can match a job trigger, but
	// nothing else.
	if w.Incoming.SelfMessage {
		return
	}
	// Check for built-in run job
	if w.isCommand {
		var jname string
		cmsg := spaceRe.ReplaceAllString(w.msg, " ")
		matches := runJobRe.FindAllStringSubmatch(cmsg, -1)
		if matches != nil {
			jname = matches[0][1]
			messageMatched = true
			w.messageHeard()
		} else {
			return
		}
		t := w.jobAvailable(jname)
		if t != nil {
			r := w.makeRobot()
			visible, jchan := r.jobVisible(t, false, false)
			if !visible {
				if len(jchan) > 0 {
					r.Say("Job not available in this channel; try: %s", jchan)
				} else {
					r.Say("Sorry, that job isn't available")
				}
				return
			}
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
				w.Say("Job '%s' is disabled: %s", jname, task.reason)
				w.deregister()
				return
			}
			if !w.jobSecurityCheck(t, "run") {
				w.deregister()
				return
			}
			var args []string
			// remember which job we're talking about
			ctx := w.makeMemoryContext("context:task")
			s := ephemeralMemory{jname, time.Now()}
			ephemeralMemories.Lock()
			ephemeralMemories.m[ctx] = s
			ephemeralMemories.Unlock()
			if len(matches[0][2]) > 0 {
				// arguments supplied with `run job foo bar baz`, check match to required arguments
				args = strings.Split(matches[0][2], " ")
				numargs := len(args)
				if numargs > 0 && numargs < len(job.Arguments) {
					w.Say("Too few arguments to job '%s', %d required but %d given", jname, len(job.Arguments), len(args))
					w.deregister()
					return
				}
				// Check regexes for required arguments
				for i, jobarg := range job.Arguments {
					if !jobarg.re.MatchString(args[i]) {
						w.Say("'%s' doesn't match the pattern for argument '%s': %s", args[i], jobarg.Label, jobarg.Regex)
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
									r.Log(robot.Warn, "failed getting arguments running job '%s': %s", jname, ret)
									r.Say("(not running job '%s')", jname)
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
			w.startPipeline(nil, t, jobCommand, "run", args...)
		} // jobAvailable sends a message if it's not
	}
	return
}
