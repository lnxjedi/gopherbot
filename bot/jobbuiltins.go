package bot

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

/*

Job builtins are special:
- They're available in every channel
- Permissions are checked against the job being operated on, not job builtin

*/

const histPageSize = 2048 // how much history to display at a time

const builtInHistoryConfig = `
AllChannels: true
AllowDirect: false
Help:
- Keywords: [ "history", "job" ]
  Helptext: [ "(bot), history <job(:namespace)> - list the histories available for a job" ]
- Keywords: [ "quit" ]
  Helptext: [ "(bot), history (job(:namespace) <run#> - start paging the history text for a job run" ]
CommandMatchers:
- Command: history
  Regex: '(?i:history(?: ([A-Za-z][\w-:]*))?)'
  Contexts: [ "task" ]
- Command: showhistory
  Regex: '(?i:history (?:([A-Za-z][\w-:]*) )?(\d+))'
  Contexts: [ "task" ]
MessageMatchers:
- Command: history
  Regex: '(?i:^history(?: ([A-Za-z][\w-:]*))?$)'
  Contexts: [ "task" ]
- Command: showhistory
  Regex: '(?i:^history (?:([A-Za-z][\w-:]*) )?(\d+)$)'
  Contexts: [ "task" ]
ReplyMatchers:
- Label: paging
  Regex: '(?i:(c|n|q))'
`

const builtInJobConfig = `
AllChannels: true
AllowDirect: false
Help:
- Keywords: [ "jobs" ]
  Helptext: [ "(bot), list (all) jobs - list the jobs you have access to, optionally in all channels" ]
CommandMatchers:
- Command: jobs
  Regex: '(?i:list (all )?jobs)'
`

func init() {
	RegisterPlugin("builtInhistory", PluginHandler{DefaultConfig: builtInHistoryConfig, Handler: jobhistory})
	RegisterPlugin("builtInJobcmd", PluginHandler{DefaultConfig: builtInJobConfig, Handler: jobcommands})
}

func jobcommands(r *Robot, command string, args ...string) (retval TaskRetVal) {
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
		c := r.getContext()
		for _, t := range c.tasks.t {
			task, _, job := getTask(t)
			if job == nil {
				continue
			}
			if !alljobs && r.Channel != task.Channel {
				continue
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
					continue
				}
			}
			after := ""
			if task.Disabled {
				after = fmt.Sprintf(" (disabled: %s)", task.reason)
			}
			if alljobs && r.Channel != task.Channel {
				jl = append(jl, fmt.Sprintf("%s (channel: %s)%s", task.name, task.Channel, after))
			} else {
				jl = append(jl, fmt.Sprintf("%s%s", task.name, after))
			}
		}
		if len(jl) == 1 {
			if alljobs {
				r.Say("I dont' have any jobs configured")
				return
			} else {
				r.Say("I don't see any jobs configured for this channel")
				return
			}
		}
		r.Say(strings.Join(jl, "\n"))
	}
	return
}

func jobhistory(r *Robot, command string, args ...string) (retval TaskRetVal) {
	if command == "init" {
		return
	}

	histSpec := args[0]
	// boilerplate availability and security checking for job commands
	c := r.getContext()
	jobName := strings.Split(histSpec, ":")[0]
	t := c.jobAvailable(jobName, true)
	if t == nil {
		return
	}
	if !c.jobSecurityCheck(t) {
		return
	}

	switch command {
	case "history":
		var jh jobHistory
		key := histPrefix + histSpec
		_, _, ret := checkoutDatum(key, &jh, false)
		if ret != Ok || len(jh.Histories) == 0 {
			r.Say(fmt.Sprintf("No history found for '%s'", histSpec))
			return
		}
		hl := make([]string, len(jh.Histories)+1)
		hl = append(hl, fmt.Sprintf("History of job runs for '%s':", histSpec))
		for _, he := range jh.Histories {
			hl = append(hl, fmt.Sprintf("Run %d - %s", he.LogIndex, he.CreateTime))
		}
		r.Say(strings.Join(hl, "\n"))
	case "showhistory":
		index, _ := strconv.Atoi(args[1])
		f, err := robot.history.GetHistory(histSpec, index)
		if err != nil {
			Log(Error, fmt.Sprintf("Error getting history %d for task '%s': %v", index, histSpec, err))
			r.Say(fmt.Sprintf("History %d for '%s' not available", index, histSpec))
			return
		}
		var line string
		scanner := bufio.NewScanner(f)
		finished := false
	PageLoop:
		for {
			size := 0
			lines := make([]string, 0, 40)
			if len(line) > 0 {
				lines = append(lines, line)
				size += len(line) + 1
				line = ""
			}
			for size < histPageSize {
				if scanner.Scan() {
					line = scanner.Text()
					size += len(line) + 1
					if size < histPageSize {
						lines = append(lines, line)
						line = ""
					}
				} else {
					finished = true
					break
				}
			}
			r.Fixed().Say(strings.Join(lines, "\n"))
			if finished {
				break
			}
			rep, ret := r.PromptForReply("paging", "'c' to continue, 'q' to quit, or 'n' to skip to the next section")
			if ret != Ok && ret != TimeoutExpired {
				r.Say("(quitting)")
				break PageLoop
			} else {
			ContinueSwitch:
				switch rep {
				case "q", "Q":
					break PageLoop
				case "n", "N":
					for scanner.Scan() {
						line = scanner.Text()
						if strings.HasPrefix(line, "***") {
							break ContinueSwitch
						}
					}
				}
			}
		}
	}
	return
}

// jobSecurityCheck performs all security checks - RequireAdmin, Authorization
// and Elevation - and returns true if passed. It will message the user and
// return false if a check fails.
func (c *botContext) jobSecurityCheck(t interface{}) bool {
	if c.automaticTask {
		return true
	}
	task, _, _ := getTask(t)
	if task.RequireAdmin {
		r := c.makeRobot()
		if !r.CheckAdmin() {
			r.Say("Sorry, that command is only available to bot administrators")
			return false
		}
	}
	if c.checkAuthorization(t, "run") != Success {
		return false
	}
	if !c.elevated {
		eret, required := c.checkElevation(t, "")
		if eret != Success {
			return false
		}
		if required {
			c.elevated = true
		}
	}
	return true
}

// jobAvailable does the work of looking up a job and checking whether it's
// available, and messaging the user if it's not. Only called for interactive
// job commands like history, run job, etc.
func (c *botContext) jobAvailable(taskName string, pluginOk bool) interface{} {
	r := c.makeRobot()
	t := c.tasks.getTaskByName(taskName)
	if t == nil {
		r.Say(fmt.Sprintf("Sorry, I don't have a task named '%s' configured", taskName))
		return nil
	}
	task, plugin, job := getTask(t)
	isPlugin := plugin != nil
	isJob := job != nil
	if !isJob && (!isPlugin && pluginOk) {
		r.Say(fmt.Sprintf("Sorry, '%s' isn't a job", taskName))
		return nil
	}
	if isPlugin {
		ok := c.pluginAvailable(task, false, false)
		if ok {
			return t
		}
		r.Say(fmt.Sprintf("Sorry, '%s' isn't available", taskName))
		return nil
	}
	if r.Channel != task.Channel {
		c.debugTask(task, fmt.Sprintf("not available in channel '%s'", task.Channel), false)
		r.Say(fmt.Sprintf("Sorry, job '%s' isn't available in this channel, try '%s'", taskName, task.Channel))
		return nil
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
			r.Say("Sorry, you're not on the list of allowed users for that job")
			c.debugTask(task, "user is not on the list of allowed users", false)
			return nil
		}
	}
	return t
}
