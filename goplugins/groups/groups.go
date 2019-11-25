// Package groups implements a groups demonstrator plugin showing how you
// can use the robot's brain to remember things - like groups of users.
package groups

import (
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/lnxjedi/gopherbot/robot"
)

const datumName = "group"

const groupHelp = `The groups plugin allows you to configure groups, members, and
 group administrators who are able to add and remove members that are
 stored in the robot's memory. For authorization purposes, any user configured
 as a member or administrator, or stored as a member in the robot's long-term
 memory, is considered a member. Note that bot administrators can also add
 and remove users from groups, but are not considered members unless explicitly
 added. 'help groups' will give help for all group related commands.`

type groupSpec struct {
	Administrators, Users []string // used with map[string]groupSpec
}

type config struct {
	Groups map[string]groupSpec
}

var spaces = regexp.MustCompile(`\s+`)

func addnew(list []string, item string) ([]string, bool) {
	add := true
	for _, listitem := range list {
		if listitem == item {
			add = false
			break
		}
	}
	if add {
		list = append(list, item)
	}
	return list, add
}

// Define the handler function
func groups(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	m := r.GetMessage()
	if command == "init" { // ignore init
		return
	}
	var cfgspec, memspec groupSpec
	var lock, group string
	var ret robot.RetVal

	groupCfg := &config{}

	ret = r.GetTaskConfig(&groupCfg)
	if ret != robot.Ok {
		r.Log(robot.Error, "Error loading groups config: %s")
		return robot.Fail
	}

	updated := false

	// Get the group name from arguments
	switch command {
	case "add", "remove", "authorize":
		group = args[1]
	case "empty", "show":
		group = args[0]
	}

	if len(group) > 0 {
		var ok bool
		// Validate the group
		cfgspec, ok = groupCfg.Groups[group]
		if !ok {
			if command != "authorize" {
				r.Say("I don't have a \"%s\" group configured", group)
				return
			}
			r.Log(robot.Warn, "Groups plugin called for non-configured group: %s", group)
			return robot.ConfigurationError
		}
	}

	if command == "authorize" && len(group) == 0 {
		r.Log(robot.Error, "Groups plugin requires a group name for authorization; plugin \"%s\" must set 'AuthRequire'", args[0])
		return robot.ConfigurationError
	}

	botAdmin := r.CheckAdmin()

	// First, check out the group map and verify admins
	switch command {
	case "help":
		r.Say(strings.Replace(groupHelp, "\n", "", -1))
		return
	case "show", "authorize":
		// read-only cases
		_, _, ret = r.CheckoutDatum(group, &memspec, false)
	case "add", "remove", "empty":
		// read-write cases, require admin privileges
		isAdmin := botAdmin
		if !isAdmin {
			if len(cfgspec.Administrators) == 0 {
				r.Log(robot.Error, "No administrators configured for group: %s", group)
			} else {
				for _, adminUser := range cfgspec.Administrators {
					if m.User == adminUser {
						isAdmin = true
						break
					}
				}
			}
		}
		if !isAdmin {
			r.Say("Sorry, only a group administrator can do that")
			return
		}
		lock, _, ret = r.CheckoutDatum(group, &memspec, true)
		defer func() {
			if !updated {
				// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
				// in case there's another thread waiting.
				r.CheckinDatum(group, lock)
			}
		}()
	}
	if ret != robot.Ok {
		r.Log(robot.Error, "Couldn't load groupspec: %s", ret)
		r.Reply("I had a problem loading the group, somebody should check my log file")
		r.CheckinDatum(group, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}

	// Now actually DO something
	switch command {
	case "remove":
		user := args[0]
		if len(memspec.Users) == 0 {
			r.Say("There are no dynamic users in the \"%s\" group", group)
			return
		}
		found := false
		for i, li := range memspec.Users {
			if user == li {
				memspec.Users[i] = memspec.Users[len(memspec.Users)-1]
				memspec.Users = memspec.Users[:len(memspec.Users)-1]
				found = true
				mret := r.UpdateDatum(group, lock, &memspec)
				if mret != robot.Ok {
					r.Log(robot.Error, "Couldn't update groups: %s", mret)
					r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
				} else {
					r.Log(robot.Audit, "User %s removed user %s from group %s", m.User, user, group)
					r.Say("Ok, I removed %s from the %s group", user, group)
					updated = true
				}
				break
			}
		}
		if !found {
			r.Say("%s isn't a dynamic member of the %s group (but may be a configured user)", user, group)
			return
		}
	case "empty":
		memspec.Users = []string{}
		mret := r.UpdateDatum(group, lock, &memspec)
		if mret != robot.Ok {
			r.Log(robot.Error, "Couldn't update groups: %s", mret)
			r.Reply("Crud. I had a problem saving the group - somebody better check the log")
		} else {
			r.Log(robot.Audit, "User %s removed all users from group %s", m.User, group)
			r.Say("Emptied")
			updated = true
		}
	case "list":
		groups := make([]string, 0, 10)
		for name, cfgspec := range groupCfg.Groups {
			add := botAdmin
			if !add {
				for _, adminUser := range cfgspec.Administrators {
					if m.User == adminUser {
						add = true
						break
					}
				}
			}
			if add {
				groups = append(groups, name)
			}
		}
		if len(groups) == 0 {
			r.Say("You're not the administrator of any groups")
			return
		}
		r.Say("Here are the groups you're an administrator for:\n%s", strings.Join(groups, "\n"))
	case "show":
		members := make([]string, 0, 10)
		for _, user := range cfgspec.Administrators {
			members, _ = addnew(members, user)
		}
		for _, user := range cfgspec.Users {
			members, _ = addnew(members, user)
		}
		for _, user := range memspec.Users {
			members, _ = addnew(members, user)
		}
		if len(members) == 0 {
			r.Say("The %s group has no members", group)
			return
		}
		r.Say("The %s group has the following members:\n%s", group, strings.Join(members, "\n"))
	case "authorize":
		isMember := false
		for _, member := range cfgspec.Administrators {
			if m.User == member {
				isMember = true
			}
		}
		for _, member := range cfgspec.Users {
			if m.User == member {
				isMember = true
			}
		}
		for _, member := range memspec.Users {
			if m.User == member {
				isMember = true
			}
		}
		if isMember {
			return robot.Success
		}
		return robot.Fail
	case "add":
		// Case sensitive input, case insensitve equality checking
		user := args[0]
		var added bool
		memspec.Users, added = addnew(memspec.Users, user)
		if added {
			mret := r.UpdateDatum(group, lock, &memspec)
			if mret != robot.Ok {
				r.Log(robot.Error, "Couldn't update groups: %s", mret)
				r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
			} else {
				updated = true
			}
			r.Log(robot.Audit, "User %s added user %s to group %s", m.User, user, group)
			r.Say("Ok, I added %s to the %s group", user, group)
		} else {
			r.Say("User %s is already in the %s group", user, group)
		}
	}
	return
}

func init() {
	bot.RegisterPlugin("groups", robot.PluginHandler{
		Handler: groups,
		Config:  &config{},
	})
}
