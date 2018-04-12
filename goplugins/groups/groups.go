// Package groups implements a groups demonstrator plugin showing how you
// can use the robot's brain to remember things - like groups of users.
package groups

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
)

const datumName = "group"

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
func groups(r *bot.Robot, command string, args ...string) (retval bot.PlugRetVal) {
	if command == "init" { // ignore init
		return
	}
	var cfgspec, memspec groupSpec
	var lock, group string
	var ret bot.RetVal

	groupCfg := &config{}

	ret = r.GetPluginConfig(&groupCfg)
	if ret != bot.Ok {
		r.Log(bot.Error, "Error loading groups config: %s")
		return bot.Fail
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
				r.Say(fmt.Sprintf("I don't have a \"%s\" group configured", group))
				return
			}
			r.Log(bot.Warn, fmt.Sprintf("Groups plugin called for non-configured group: %s", group))
			return bot.ConfigurationError
		}
	}

	if command == "authorize" && len(group) == 0 {
		r.Log(bot.Error, fmt.Sprintf("Groups plugin requires a group name for authorization; plugin \"%s\" must set 'AuthRequire'", args[0]))
		return bot.ConfigurationError
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
				r.Log(bot.Error, fmt.Sprintf("No administrators configured for group: %s", group))
			} else {
				for _, adminUser := range cfgspec.Administrators {
					if r.User == adminUser {
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
	if ret != bot.Ok {
		r.Log(bot.Error, fmt.Sprintf("Couldn't load groupspec: %s", ret))
		r.Reply("I had a problem loading the group, somebody should check my log file")
		r.CheckinDatum(group, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}

	// Now actually DO something
	switch command {
	case "remove":
		user := args[0]
		if len(memspec.Users) == 0 {
			r.Say(fmt.Sprintf("There are no dynamic users in the \"%s\" group", group))
			return
		}
		found := false
		for i, li := range memspec.Users {
			if user == li {
				memspec.Users[i] = memspec.Users[len(memspec.Users)-1]
				memspec.Users = memspec.Users[:len(memspec.Users)-1]
				found = true
				mret := r.UpdateDatum(group, lock, &memspec)
				if mret != bot.Ok {
					r.Log(bot.Error, fmt.Sprintf("Couldn't update groups: %s", mret))
					r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
				} else {
					r.Say(fmt.Sprintf("Ok, I removed %s from the %s group", user, group))
					updated = true
				}
				break
			}
		}
		if !found {
			r.Say(fmt.Sprintf("%s isn't a dynamic member of the %s group (but may be a configured user)", user, group))
			return
		}
	case "empty":
		memspec.Users = []string{}
		mret := r.UpdateDatum(group, lock, &memspec)
		if mret != bot.Ok {
			r.Log(bot.Error, fmt.Sprintf("Couldn't update groups: %s", mret))
			r.Reply("Crud. I had a problem saving the group - somebody better check the log")
		} else {
			r.Say("Emptied")
			updated = true
		}
	case "list":
		groups := make([]string, 0, 10)
		for name, cfgspec := range groupCfg.Groups {
			add := botAdmin
			if !add {
				for _, adminUser := range cfgspec.Administrators {
					if r.User == adminUser {
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
		r.Say(fmt.Sprintf("Here are the groups you're an administrator for:\n%s", strings.Join(groups, "\n")))
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
			r.Say(fmt.Sprintf("The %s group has no members", group))
			return
		}
		r.Say(fmt.Sprintf("The %s group has the following members:\n%s", group, strings.Join(members, "\n")))
	case "authorize":
		isMember := false
		for _, member := range cfgspec.Administrators {
			if r.User == member {
				isMember = true
			}
		}
		for _, member := range cfgspec.Users {
			if r.User == member {
				isMember = true
			}
		}
		for _, member := range memspec.Users {
			if r.User == member {
				isMember = true
			}
		}
		if isMember {
			return bot.Success
		}
		return bot.Fail
	case "add":
		// Case sensitive input, case insensitve equality checking
		user := args[0]
		var added bool
		memspec.Users, added = addnew(memspec.Users, user)
		if added {
			mret := r.UpdateDatum(group, lock, &memspec)
			if mret != bot.Ok {
				r.Log(bot.Error, fmt.Sprintf("Couldn't update groups: %s", mret))
				r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
			} else {
				updated = true
			}
			r.Say(fmt.Sprintf("Ok, I added %s to the %s group", user, group))
		} else {
			r.Say(fmt.Sprintf("User %s is already in the %s group", user, group))
		}
	}
	return
}

func init() {
	bot.RegisterPlugin("groups", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       groups,
		Config:        &config{},
	})
}
