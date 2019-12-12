package bot

import (
	"fmt"
	"path/filepath"
	"strings"
)

// pluginAvailable checks the user and channel against the task's
// configuration to determine if the task should be available. Used by
// both handleMessage and the help builtin. verboseOnly is set when availability
// is being checked for ambient messages or auth/elevation plugins, to indicate
// debugging verboseness.
func (c *botContext) pluginAvailable(task *Task, helpSystem, verboseOnly bool) (available bool) {
	nvmsg := "task is NOT visible to user " + c.User + " in channel "
	vmsg := "task is visible to user " + c.User + " in channel "
	if c.directMsg {
		nvmsg += "(direct message)"
		vmsg += "(direct message)"
	} else {
		nvmsg += c.Channel
		vmsg += c.Channel
	}
	defer func(vmsg string) {
		if available {
			c.debugTask(task, vmsg, verboseOnly)
		}
	}(vmsg)
	if task.Disabled {
		c.debugTask(task, nvmsg+"; task is disabled, possibly due to configuration error", verboseOnly)
		return false
	}
	if !c.directMsg && task.DirectOnly && !helpSystem {
		c.debugTask(task, nvmsg+"; only available by direct message: DirectOnly is TRUE", verboseOnly)
		return false
	}
	if c.directMsg && !task.AllowDirect && !helpSystem {
		c.debugTask(task, nvmsg+"; not available by direct message: AllowDirect is FALSE", verboseOnly)
		return false
	}
	if task.RequireAdmin {
		isAdmin := false
		admins := c.cfg.adminUsers
		for _, adminUser := range admins {
			if c.User == adminUser {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			c.debugTask(task, nvmsg+"; RequireAdmin is TRUE and user isn't an Admin", verboseOnly)
			return false
		}
	}
	if len(task.Users) > 0 {
		userOk := false
		for _, allowedUser := range task.Users {
			match, err := filepath.Match(allowedUser, c.User)
			if match && err == nil {
				userOk = true
			}
		}
		if !userOk {
			c.debugTask(task, nvmsg+"; user is not on the list of allowed users", verboseOnly)
			return false
		}
	}
	if c.directMsg && (task.AllowDirect || task.DirectOnly) {
		return true
	}
	if len(task.Channels) > 0 {
		for _, pchannel := range task.Channels {
			if pchannel == c.Channel {
				return true
			}
		}
	} else {
		if task.AllChannels {
			return true
		}
	}
	if helpSystem {
		return true
	}
	c.debugTask(task, fmt.Sprintf(nvmsg+"; channel '%s' is not on the list of allowed channels: %s", c.Channel, strings.Join(task.Channels, ", ")), verboseOnly)
	return false
}
