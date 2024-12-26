package bot

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

/*

Job builtins are special:
- They're available in every channel
- Permissions are checked against the job being operated on, not job builtin

*/

const histPageSize = 2048 // how much history to display at a time

func init() {
	robot.RegisterPlugin("builtin-history", robot.PluginHandler{Handler: jobhistory})
	robot.RegisterPlugin("builtin-jobcmd", robot.PluginHandler{Handler: jobcommands})
}

func jobcommands(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	tasks := r.tasks
	if command == "init" {
		return
	}
	switch command {
	case "jobs":
		var jl []string
		alljobs := len(args[0]) > 0
		if alljobs {
			jl = []string{"Here's a list of all the jobs I know about:"}
		} else {
			jl = []string{"Here's a list of jobs for this channel:"}
		}
		for _, t := range tasks.t[1:] {
			if ok, _ := r.jobVisible(t, alljobs, true); !ok {
				continue
			}
			task, _, _ := getTask(t)
			after := ""
			if task.Disabled {
				after = fmt.Sprintf(" (disabled: %s)", task.reason)
			}
			if alljobs {
				jl = append(jl, fmt.Sprintf("%s (channel: %s)%s", task.name, task.Channel, after))
			} else {
				jl = append(jl, fmt.Sprintf("%s%s", task.name, after))
			}
		}
		if len(jl) == 1 {
			if alljobs {
				r.Say("I dont' have any jobs configured")
				return
			}
			r.Say("I don't see any jobs configured for this channel")
			return
		}
		r.Say(strings.Join(jl, "\n"))
	}
	return
}

func emailhistory(r Robot, user, address, spec string, run int) (retval robot.TaskRetVal) {
	var buff []byte
	retval, buff = getLogMail(spec, run)
	if retval != robot.Normal {
		r.Say("Log for '%s', run %d not available", spec, run)
		return
	}
	body := new(bytes.Buffer)
	body.Write([]byte("<pre>\n"))
	body.Write(buff)
	body.Write([]byte("\n</pre>"))
	subject := fmt.Sprintf("Log for pipeline '%s', run %d", spec, run)
	var ret robot.RetVal
	if len(user) > 0 {
		ret = r.EmailUser(user, subject, body, true)
	} else if len(address) > 0 {
		ret = r.EmailAddress(address, subject, body, true)
	} else {
		ret = r.Email(subject, body, true)
	}
	if ret != robot.Ok {
		r.Reply("There was a problem emailing the history log, contact an administrator")
		return
	}
	r.Say("Email sent")
	return
}

func jobhistory(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	if command == "init" {
		return
	}
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	w.Unlock()

	var histRef, histSpec, jobName, index, user, address string
	var idx int

	switch command {
	case "maillog":
		histRef = args[0]
		user = args[1]
		address = args[2]
	case "taillog", "linklog":
		histRef = args[0]
	case "joblogs":
		jobName = args[0]
	}

	if len(index) > 0 {
		var err error
		idx, err = strconv.Atoi(index)
		if err != nil {
			r.Say("Unable to convert '%s' to an index", index)
			return
		}
	}

	if len(histRef) > 0 {
		lmap := make(map[string]historyLookup)
		_, _, lret := checkoutDatum(histLookup, &lmap, false)
		if lret != robot.Ok {
			r.Say("There was a memory error looking up that log")
			r.Log(robot.Error, "Looking up '%s': %s", histLookup, lret)
			return
		}
		hl, ok := lmap[histRef]
		if !ok {
			r.Say("Log ref '%s' not found, possibly expired?", histRef)
			r.Log(robot.Warn, "Log ref '%s' not found: %s", histRef)
			return
		}
		histSpec = hl.Tag
		idx = hl.Index
	}

	if len(jobName) > 0 {
		histSpec = jobName
	}

	// boilerplate availability and security checking for job commands;
	// note that both jobAvailable and jobSecurityCheck emit messages
	// to the user if they fail
	jname := strings.Split(histSpec, ":")[0]
	t := w.jobAvailable(jname)
	if t == nil {
		return
	}
	if !w.jobSecurityCheck(t, command) {
		return
	}

	jh := pipeHistory{}
	key := histPrefix + histSpec
	_, _, ret := checkoutDatum(key, &jh, false)
	if ret != robot.Ok {
		r.Say("No logs found for '%s'", histSpec)
		return
	}
	if len(jh.Histories) == 0 {
		r.Say("No logs found for '%s'", histSpec)
		return
	}

	switch command {
	case "maillog":
		return emailhistory(r, user, address, histSpec, idx)
	case "taillog":
		ret, buff := getLogTail(histSpec, idx)
		if ret == robot.Normal {
			r.Say("log excerpt for '%s', run %d:", histSpec, idx)
			r.Fixed().Say(string(buff))
		} else {
			r.Say("Log for '%s', run %d not available", histSpec, idx)
		}
	case "linklog":
		url, found := interfaces.history.GetLogURL(histSpec, idx)
		if !found {
			url, found = interfaces.history.MakeLogURL(histSpec, idx)
		}
		if !found {
			r.Say("URL for '%s', run %d not available", histSpec, idx)
			return
		}
		r.Say("Here you go: %s", url)
	case "joblogs":
		var loglines []string
		loglines = []string{fmt.Sprintf("Logs for job '%s':", jobName)}
		for _, log := range jh.Histories {
			var logline string
			logline = fmt.Sprintf("Run #%d, %s; log %s", log.LogIndex, log.CreateTime, log.Ref)
			loglines = append(loglines, logline)
		}
		r.Say(strings.Join(loglines, "\n"))
		return
	}
	return
}

// jobSecurityCheck performs all security checks - RequireAdmin, Authorization
// and Elevation - and returns true if passed. It will message the user and
// return false if a check fails.
func (w *worker) jobSecurityCheck(t interface{}, command string) bool {
	if w.automaticTask {
		return true
	}
	if w.Incoming.HiddenMessage {
		w.Reply("Sorry, job commands cannot be run as hidden commands - use the robot's name or alias")
		return false
	}
	task, _, _ := getTask(w.currentTask)
	if task.RequireAdmin {
		if !w.checkAdmin() {
			w.Say("Sorry, that command is only available to bot administrators")
			return false
		}
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	if r.checkAuthorization(w, t, command) != robot.Success {
		deregisterWorker(r.tid)
		return false
	}
	if !w.elevated {
		eret, _ := r.checkElevation(t, command)
		if eret != robot.Success {
			deregisterWorker(r.tid)
			return false
		}
	}
	deregisterWorker(r.tid)
	return true
}

// jobVisible checks whether a user should see a job in a channel, unless
// ignoreChannelRestrictions is set. Note that changes to logic in jobVisible
// may need to propagate to jobAvailable, below.
func (r Robot) jobVisible(t interface{}, ignoreChannelRestrictions, disabledOk bool) (visible bool, channel string) {
	task, _, job := getTask(t)
	if job == nil {
		return
	}
	if task.Disabled && !disabledOk {
		return
	}
	if len(task.Users) > 0 {
		userOk := false
		for _, allowedUser := range task.Users {
			match, err := filepath.Match(allowedUser, r.User)
			if match && err == nil {
				userOk = true
			}
		}
		if !userOk {
			return
		}
	}
	if task.RequireAdmin {
		isAdmin := false
		admins := r.cfg.adminUsers
		for _, adminUser := range admins {
			if r.User == adminUser {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			return
		}
	}
	if !ignoreChannelRestrictions && r.Channel != task.Channel {
		channel = task.Channel
		return
	}
	return true, ""
}

// jobAvailable does the work of looking up a job and checking whether it's
// available, and messaging the user if it's not. Only called for interactive
// job commands like history, run job, etc. where the user provides a job name.
// Note that changes to login in jobAvailable may need to propagate to
// jobVisible, above.
func (w *worker) jobAvailable(taskName string) interface{} {
	t := w.tasks.getTaskByName(taskName)
	if t == nil {
		w.Say("Sorry, I couldn't find a job named '%s' configured", taskName)
		return nil
	}
	task, _, job := getTask(t)
	isJob := job != nil
	if !isJob {
		w.Say("Sorry, '%s' isn't a job", taskName)
		return nil
	}
	if w.automaticTask {
		return t
	}
	// If there's already a job initialized, this is a pipeline task for that
	// job, and should be available regardless of channel.
	if w.pipeContext != nil && !w.jobInitialized && w.Channel != task.Channel {
		w.Say("Sorry, job '%s' isn't available in this channel, try '%s'", taskName, task.Channel)
		return nil
	}
	if task.RequireAdmin {
		isAdmin := false
		admins := w.cfg.adminUsers
		for _, adminUser := range admins {
			if w.User == adminUser {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			w.Say("Sorry, '%s' is only available to bot administrators", taskName)
			return nil
		}
	}
	if len(task.Users) > 0 {
		userOk := false
		for _, allowedUser := range task.Users {
			match, err := filepath.Match(allowedUser, w.User)
			if match && err == nil {
				userOk = true
			}
		}
		if !userOk {
			w.Say("Sorry, you're not on the list of allowed users for that job")
			return nil
		}
	}
	return t
}
