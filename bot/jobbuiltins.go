package bot

import (
	"fmt"
	"path/filepath"
)

/*

Job builtins are special:
- They're available in every channel
- Permissions are checked against the job being operated on, not job builtin

*/

const builtInHistoryConfig = `
AllChannels: true
AllowDirect: false
Help:
- Keywords: [ "history", "job" ]
  Helptext: [ "(bot), history <job> - list the histories available for a job" ]
- Keywords: [ "quit" ]
  Helptext: [ "(bot), history <job> <run#> - start paging the history text for a job run" ]
CommandMatchers:
- Command: history
  Regex: '(?i:history ([\w-]+))'
- Command: showhistory
  Regex: '(?i:history ([\w-]+) (\d+))'
`

func init() {
	RegisterPlugin("builtInhistory", PluginHandler{DefaultConfig: builtInHistoryConfig, Handler: jobhistory})
}

func jobhistory(r *Robot, command string, args ...string) (retval TaskRetVal) {
	if command == "init" {
		return
	}

	// boilerplate availability and security checking for job commands
	taskName := args[0]
	t := r.jobAvailable(taskName)
	if t == nil {
		return
	}
	c := r.getContext()
	if !c.jobSecurityCheck(t) {
		return
	}

	switch command {
	case "history":

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
func (r *Robot) jobAvailable(taskName string) interface{} {
	c := r.getContext()
	t := c.tasks.getTaskByName(taskName)
	if t == nil {
		r.Say(fmt.Sprintf("Sorry, I don't have a task named '%s' configured", taskName))
		return nil
	}
	task, _, job := getTask(t)
	if job == nil {
		r.Say(fmt.Sprintf("Sorry, '%s' isn't a job", taskName))
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
