package bot

import (
	"fmt"
	"regexp"
)

const runJobRegex = `run +job +(` + identifierRegex + `)`
const historyRegex = `history +(` + identifierRegex + `)`
const showHistoryRegex = `history +(` + identifierRegex + `) +(\d+)`

var runJobRe = regexp.MustCompile(`(?i:^\s*` + runJobRegex + `\s*$)`)
var historyRe = regexp.MustCompile(`(?i:^\s*` + historyRegex + `\s*$)`)
var showHistoryRe = regexp.MustCompile(`(?i:^\s*` + showHistoryRegex + `\s*$)`)

type jobCommand int

const (
	jobRun = iota
	history
	showHistory
)

// checkJobMatchersAndRun handles triggers, 'run job <foo>', 'history <foo>'
func (bot *botContext) checkJobMatchersAndRun() (messageMatched bool) {
	r := bot.makeRobot()
	// un-needed, but more clear
	messageMatched = false
	var runTask interface{}
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
			bot.debugTask(t, msg, false)
			continue
		}
		Log(Trace, fmt.Sprintf("Checking triggers for job '%s'", task.name))
		triggers := job.Triggers
		bot.debugTask(t, fmt.Sprintf("Checking %d JobTriggers against message: '%s' from user '%s' in channel '%s'", len(triggers), bot.msg, bot.User, bot.Channel), false)
		for _, trigger := range triggers {
			Log(Trace, fmt.Sprintf("Checking '%s' against user '%s', channel '%s', regex: '%s'", bot.msg, trigger.User, trigger.Channel, trigger.Regex))
			if bot.User != trigger.User {
				bot.debugTask(t, fmt.Sprintf("User '%s' doesn't match trigger user '%s'", bot.User, trigger.User), false)
				continue
			}
			if bot.Channel != trigger.Channel {
				bot.debugTask(t, fmt.Sprintf("Channel '%s' doesn't match trigger", bot.Channel), false)
				continue
			}
			matches := trigger.re.FindAllStringSubmatch(bot.msg, -1)
			matched := false
			if matches != nil {
				bot.debugTask(t, fmt.Sprintf("Matched trigger regex '%s'", trigger.Regex), false)
				Log(Trace, fmt.Sprintf("Message '%s' matches trigger for job '%s'", bot.msg, task.name))
				matched = true
				bot.User = task.User
				triggerArgs = matches[0][1:]
			} else {
				bot.debugTask(t, fmt.Sprintf("Not matched: %s", trigger.Regex), false)
			}
			if matched {
				if messageMatched {
					prevTask, _, _ := getTask(runTask)
					Log(Error, fmt.Sprintf("Message '%s' from user '%s' in channel '%s' matched triggers for multiple jobs: '%s' and '%s', ignoring", bot.msg, bot.User, bot.Channel, prevTask.name, task.name))
					emit(MultipleMatchesNoAction)
					return
				}
				messageMatched = true
				runTask = t
				bot.Channel = task.Channel
				break
			}
		} // end of triggerer checking
	} // end of job trigger checking
	if messageMatched {
		r.messageHeard()
		robot.RLock()
		if robot.shuttingDown {
			r.Say("Ignoring triggered job: shutting down")
			robot.RUnlock()
			return
		} else if robot.paused {
			r.Say("Ignoring triggered job: paused")
			robot.RUnlock()
			return
		}
		robot.RUnlock()
		// Jobs triggers should only match apps / bots, not real users!
		bot.bypassSecurityChecks = true
		bot.startPipeline(runTask, false, jobTrigger, "run", triggerArgs...)
		return
	}
	// Check for built-in job commands
	var jobCommand jobCommand
	var command string
	var args []string
	var jobName string
	var runIndex string
	matches := runJobRe.FindAllStringSubmatch(bot.msg, -1)
	if matches != nil {
		jobCommand = jobRun
		jobName = matches[0][1]
		messageMatched = true
	}
	if !messageMatched {
		matches = historyRe.FindAllStringSubmatch(bot.msg, -1)
		if matches != nil {
			jobCommand = history
			jobName = matches[0][1]
			messageMatched = true
		}
	}
	if !messageMatched {
		matches = showHistoryRe.FindAllStringSubmatch(bot.msg, -1)
		if matches != nil {
			jobCommand = showHistory
			jobName = matches[0][1]
			runIndex = matches[0][2]
			messageMatched = true
		}
	}
	if !messageMatched {
		return
	}
	t := bot.tasks.getTaskByName(jobName)
	if t == nil {
		bot.makeRobot().Say(fmt.Sprintf("Sorry, I don't have a task named '%s' configured", jobName))
		return false
	}
	task, _, job := getTask(t)
	if job == nil {
		bot.makeRobot().Say(fmt.Sprintf("Sorry, I don't have a job named '%s' configured", jobName))
		return false
	}
	if bot.Channel != task.Channel {
		bot.makeRobot().Say(fmt.Sprintf("Sorry, job '%s' isn't available in this channel, try '%s'", jobName, task.Channel))
		return false
	}
	switch jobCommand {
	case jobRun:
		// TODO: prompt for required parameters & add to bot.environment
		// NOTE: 'run job' uses security checks in startPipeline
		bot.startPipeline(t, true, jobCmd, "run")
		return
	case history, showHistory:
		runTask = bot.tasks.getTaskByName("builtInhistory")
		if len(runIndex) > 0 {
			command = "showhistory"
		} else {
			command = "history"
		}
		args = []string{jobName, runIndex}
	}
	// NOTE: other job commands need security checks for the job being operated on
	if bot.checkAuthorization(t, "run") != Success {
		bot.makeRobot().Say(fmt.Sprintf("Sorry, failed authorization for job '%s'", jobName))
		return
	}
	if !bot.elevated {
		eret, required := bot.checkElevation(t, command)
		if eret != Success {
			bot.makeRobot().Say("Elevation request failed")
			return
		}
		if required {
			bot.elevated = true
		}
	}
	bot.startPipeline(runTask, true, jobCmd, command, args...)
	return
}
