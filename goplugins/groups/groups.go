// Package groups implements a groups demonstrator plugin showing how you
// can use the robot's brain to remember things - like groups of items.
package groups

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
)

const datumName = "groupmap"

var spaces = regexp.MustCompile(`\s+`)

type itemGroup []string

type config struct {
	Scope string
}

// Define the handler function
func groups(r *bot.Robot, command string, args ...string) (retval bot.PlugRetVal) {
	// Create an empty map to unmarshal into
	if command == "init" { // ignore init
		return
	}
	var groups = make(map[string]itemGroup)
	var lock string
	var ret bot.RetVal
	scope := &config{}

	datumKey := datumName // default global
	ret = r.GetPluginConfig(&scope)
	r.Log(bot.Debug, fmt.Sprintf("Retrieved groups config: %v", scope))
	if ret == bot.Ok {
		if strings.ToLower(scope.Scope) == "channel" {
			datumKey = r.Channel + ":" + datumName
		}
	}

	updated := false
	// First, check out the group
	switch command {
	case "help":
		r.Say(strings.Replace(groupHelp, "\n", "", -1))
		return
	case "show", "send", "pick", "group":
		// read-only cases
		_, _, ret = r.CheckoutDatum(datumKey, &groups, false)
	default:
		// all other cases are read-write
		lock, _, ret = r.CheckoutDatum(datumKey, &groups, true)
		defer func() {
			if !updated {
				// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
				// in case there's another thread waiting.
				r.CheckinDatum(datumKey, lock)
			}
		}()
	}
	if ret != bot.Ok {
		r.Log(bot.Error, fmt.Sprintf("Couldn't load groups: %s", ret))
		r.Reply("I had a problem loading the groups, somebody should check my log file")
		r.CheckinDatum(datumKey, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}
	switch command {
	case "remove":
		item := args[0]
		groupName := strings.ToLower((args[1]))
		group, ok := groups[groupName]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a group named %s", args[1]))
			return
		}
		citem := strings.ToLower(item)
		found := false
		for i, li := range group {
			if citem == strings.ToLower(li) {
				group[i] = group[len(group)-1]
				group = group[:len(group)-1]
				groups[groupName] = group
				r.Say(fmt.Sprintf("Ok, I removed %s from the %s group", item, groupName))
				found = true
				mret := r.UpdateDatum(datumKey, lock, groups)
				if mret != bot.Ok {
					r.Log(bot.Error, fmt.Sprintf("Couldn't update groups: %s", mret))
					r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
				} else {
					updated = true
				}
				break
			}
		}
		if !found {
			r.Say(fmt.Sprintf("I didn't see %s on the %s group", item, groupName))
			return
		}
	case "empty", "delete":
		groupName := strings.ToLower(args[0])
		_, ok := groups[groupName]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a group named %s", args[0]))
			return
		}
		if command == "empty" {
			groups[groupName] = []string{}
			r.Say("Emptied")
		} else {
			delete(groups, groupName)
			r.Say("Deleted")
		}
		mret := r.UpdateDatum(datumKey, lock, groups)
		if mret != bot.Ok {
			r.Log(bot.Error, fmt.Sprintf("Couldn't update groups: %s", mret))
			r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
		} else {
			updated = true
		}
	case "group":
		groupgroup := make([]string, 0, 10)
		for l := range groups {
			groupgroup = append(groupgroup, l)
		}
		if len(groupgroup) == 0 {
			if scope.Scope == "channel" {
				r.Say("I don't have any groups for this channel")
			} else {
				r.Say("I don't have any groups")
			}
			return
		}
		if scope.Scope == "channel" {
			r.Say(fmt.Sprintf("Here are the groups I have for this channel:\n%s", strings.Join(groupgroup, "\n")))
		} else {
			r.Say(fmt.Sprintf("Here are the groups I know about:\n%s", strings.Join(groupgroup, "\n")))
		}
	case "show", "send":
		groupName := strings.ToLower(args[0])
		var groupBuffer bytes.Buffer
		group, ok := groups[groupName]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a group named %s", args[0]))
			return
		}
		if len(group) == 0 {
			r.Say(fmt.Sprintf("The %s group is empty", args[0]))
			return
		}
		lineEnd := "\n"
		if command == "send" {
			lineEnd = "\r\n"
		}
		for _, li := range group {
			fmt.Fprintf(&groupBuffer, "%s%s", li, lineEnd)
		}
		switch command {
		case "show":
			r.Say(fmt.Sprintf("Here's what I have on the %s group:\n%s", groupName, strings.Trim(groupBuffer.String(), "\n")))
		case "send":
			if ret := r.Email(fmt.Sprintf("The %s group", args[0]), &groupBuffer); ret != bot.Ok {
				r.Say("Sorry, there was an error sending the email - have somebody check the my log file")
				return
			}
			botmail := r.GetBotAttribute("email").String()
			r.Say(fmt.Sprintf("Ok, I sent the %s group to you - look for email from %s", args[0], botmail))
		}
	case "pick":
		groupName := strings.ToLower(args[0])
		group, ok := groups[groupName]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a group named %s", groupName))
			return
		}
		if len(group) == 0 {
			r.Say(fmt.Sprintf("The %s group is empty", groupName))
			return
		}
		item := r.RandomString(group)
		r.RememberContext("item", item)
		r.Say(fmt.Sprintf("Here you go: %s", item))
	case "add":
		// Case sensitive input, case insensitve equality checking
		item := args[0]
		groupName := strings.ToLower(args[1])
		group, ok := groups[groupName]
		if !ok {
			r.CheckinDatum(datumKey, lock)
			rep, ret := r.PromptForReply("YesNo", fmt.Sprintf("I don't have a \"%s\" group, do you want to create it?", args[1]))
			if ret == bot.Ok {
				switch strings.ToLower(rep) {
				case "n", "no":
					r.Say("Item not added")
					return
				default:
					lock, _, ret = r.CheckoutDatum(datumKey, &groups, true)
					// Need to make sure the group wasn't created while waiting for an answer
					group, ok := groups[groupName]
					if !ok {
						groups[groupName] = []string{item}
						mret := r.UpdateDatum(datumKey, lock, groups)
						if mret != bot.Ok {
							r.Log(bot.Error, fmt.Sprintf("Couldn't update groups: %s", mret))
							r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
						} else {
							r.Say(fmt.Sprintf("Ok, I created a new %s group and added %s to it", args[1], item))
							updated = true
						}
					} else { // wow, it WAS created while waiting
						citem := strings.ToLower(item)
						for _, li := range group {
							if citem == strings.ToLower(li) {
								r.Say(fmt.Sprintf("Somebody already created the %s group and added %s to it", args[1], item))
								return
							}
						}
						group = append(group, item)
						groups[groupName] = group
						mret := r.UpdateDatum(datumKey, lock, groups)
						if mret != bot.Ok {
							r.Log(bot.Error, fmt.Sprintf("Couldn't update groups: %s", mret))
							r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
						} else {
							updated = true
						}
						r.Say(fmt.Sprintf("Ok, I added %s to the new %s group", item, args[1]))
					}
				}
			} else {
				r.Reply("Sorry, I didn't get an answer I understand")
				return
			}
		} else {
			citem := strings.ToLower(item)
			for _, li := range group {
				if citem == strings.ToLower(li) {
					r.Say(fmt.Sprintf("%s is already on the %s group", item, args[1]))
					return
				}
			}
			group = append(group, item)
			groups[groupName] = group
			mret := r.UpdateDatum(datumKey, lock, groups)
			if mret != bot.Ok {
				r.Log(bot.Error, fmt.Sprintf("Couldn't update groups: %s", mret))
				r.Reply("Crud. I had a problem saving my groups - somebody better check the log")
			} else {
				updated = true
			}
			r.Say(fmt.Sprintf("Ok, I added %s to the %s group", item, args[1]))
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
