// Package lists implements a lists demonstrator plugin showing how you
// can use the robot's brain to remember things - like lists of items.
package lists

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/lnxjedi/gopherbot/robot"
)

const datumName = "listmap"
const listHelp = `The list plugin allows you to manage simple lists of items,
 such as a todo list, lunch spots, etc. Lists default to global scope, but can be
 configured with per-channel scope. Note that 'lists' is a context aware plugin, and
 will remember the list or item being discussed; so e.g. you can follow 'add milk
 to the grocery list' with 'add hamburgers to the list' and the robot will
 know you mean the grocery list; also, 'add it to the dinner list' would add
 hamburgers to the dinner list. List names are always lowercased, but items are
 stored with case preserved and compared in a case-insensitive manner.`

var spaces = regexp.MustCompile(`\s+`)

type itemList []string

type config struct {
	Scope string
}

// Define the handler function
func lists(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	// Create an empty map to unmarshal into
	if command == "init" { // ignore init
		return
	}
	m := r.GetMessage()
	var lists = make(map[string]itemList)
	var lock string
	var ret robot.RetVal
	scope := &config{}

	datumKey := datumName // default global
	ret = r.GetTaskConfig(&scope)
	r.Log(robot.Debug, "Retrieved lists config: %v", scope)
	if ret == robot.Ok {
		if strings.ToLower(scope.Scope) == "channel" {
			datumKey = m.Channel + ":" + datumName
		}
	}

	updated := false
	// First, check out the list
	switch command {
	case "help":
		r.Say(strings.Replace(listHelp, "\n", "", -1))
		return
	case "show", "send", "pick", "list":
		// read-only cases
		_, _, ret = r.CheckoutDatum(datumKey, &lists, false)
	default:
		// all other cases are read-write
		lock, _, ret = r.CheckoutDatum(datumKey, &lists, true)
		defer func() {
			if !updated {
				// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
				// in case there's another thread waiting.
				r.CheckinDatum(datumKey, lock)
			}
		}()
	}
	if ret != robot.Ok {
		r.Log(robot.Error, "Couldn't load lists: %s", ret)
		r.Reply("I had a problem loading the lists, somebody should check my log file")
		r.CheckinDatum(datumKey, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}
	switch command {
	case "remove":
		item := args[0]
		listName := strings.ToLower((args[1]))
		list, ok := lists[listName]
		if !ok {
			r.Say("I don't have a list named %s", args[1])
			return
		}
		citem := strings.ToLower(item)
		found := false
		for i, li := range list {
			if citem == strings.ToLower(li) {
				list[i] = list[len(list)-1]
				list = list[:len(list)-1]
				lists[listName] = list
				found = true
				mret := r.UpdateDatum(datumKey, lock, lists)
				if mret != robot.Ok {
					r.Log(robot.Error, "Couldn't update lists: %s", mret)
					r.Reply("Crud. I had a problem saving my lists - somebody better check the log")
				} else {
					r.Say("Ok, I removed %s from the %s list", item, listName)
					updated = true
				}
				break
			}
		}
		if !found {
			r.Say("I didn't see %s on the %s list", item, listName)
			return
		}
	case "empty", "delete":
		listName := strings.ToLower(args[0])
		var msg string
		_, ok := lists[listName]
		if !ok {
			r.Say("I don't have a list named %s", args[0])
			return
		}
		if command == "empty" {
			lists[listName] = []string{}
			msg = "Emptied"
		} else {
			delete(lists, listName)
			msg = "Deleted"
		}
		mret := r.UpdateDatum(datumKey, lock, lists)
		if mret != robot.Ok {
			r.Log(robot.Error, "Couldn't update lists: %s", mret)
			r.Reply("Crud. I had a problem saving my lists - somebody better check the log")
		} else {
			r.Say(msg)
			updated = true
		}
	case "list":
		listlist := make([]string, 0, 10)
		for l := range lists {
			listlist = append(listlist, l)
		}
		if len(listlist) == 0 {
			if scope.Scope == "channel" {
				r.Say("I don't have any lists for this channel")
			} else {
				r.Say("I don't have any lists")
			}
			return
		}
		if scope.Scope == "channel" {
			r.Say("Here are the lists I have for this channel:\n%s", strings.Join(listlist, "\n"))
		} else {
			r.Say("Here are the lists I know about:\n%s", strings.Join(listlist, "\n"))
		}
	case "show", "send":
		listName := strings.ToLower(args[0])
		var listBuffer bytes.Buffer
		list, ok := lists[listName]
		if !ok {
			r.Say("I don't have a list named %s", args[0])
			return
		}
		if len(list) == 0 {
			r.Say("The %s list is empty", args[0])
			return
		}
		lineEnd := "\n"
		if command == "send" {
			lineEnd = "\r\n"
		}
		for _, li := range list {
			fmt.Fprintf(&listBuffer, "%s%s", li, lineEnd)
		}
		switch command {
		case "show":
			r.Say("Here's what I have on the %s list:\n%s", listName, strings.Trim(listBuffer.String(), "\n"))
		case "send":
			if ret := r.Email(fmt.Sprintf("The %s list", args[0]), &listBuffer); ret != robot.Ok {
				r.Say("Sorry, there was an error sending the email - have somebody check the my log file")
				return
			}
			botmail := r.GetBotAttribute("email").String()
			r.Say("Ok, I sent the %s list to you - look for email from %s", args[0], botmail)
		}
	case "pick":
		listName := strings.ToLower(args[0])
		list, ok := lists[listName]
		if !ok {
			r.Say("I don't have a list named %s", listName)
			return
		}
		if len(list) == 0 {
			r.Say("The %s list is empty", listName)
			return
		}
		item := r.RandomString(list)
		r.RememberContext("item", item)
		r.Say("Here you go: %s", item)
	case "add":
		// Case sensitive input, case insensitve equality checking
		item := args[0]
		listName := strings.ToLower(args[1])
		list, ok := lists[listName]
		if !ok {
			r.CheckinDatum(datumKey, lock)
			rep, ret := r.PromptForReply("YesNo", "I don't have a '%s' list, do you want to create it?", args[1])
			if ret == robot.Ok {
				switch strings.ToLower(rep) {
				case "n", "no":
					r.Say("Item not added")
					return
				default:
					lock, _, ret = r.CheckoutDatum(datumKey, &lists, true)
					// Need to make sure the list wasn't created while waiting for an answer
					list, ok := lists[listName]
					if !ok {
						lists[listName] = []string{item}
						mret := r.UpdateDatum(datumKey, lock, lists)
						if mret != robot.Ok {
							r.Log(robot.Error, "Couldn't update lists: %s", mret)
							r.Reply("Crud. I had a problem saving my lists - somebody better check the log")
						} else {
							r.Say("Ok, I created a new %s list and added %s to it", args[1], item)
							updated = true
						}
					} else { // wow, it WAS created while waiting
						citem := strings.ToLower(item)
						for _, li := range list {
							if citem == strings.ToLower(li) {
								r.Say("Somebody already created the %s list and added %s to it", args[1], item)
								return
							}
						}
						list = append(list, item)
						lists[listName] = list
						mret := r.UpdateDatum(datumKey, lock, lists)
						if mret != robot.Ok {
							r.Log(robot.Error, "Couldn't update lists: %s", mret)
							r.Reply("Crud. I had a problem saving my lists - somebody better check the log")
						} else {
							updated = true
						}
						r.Say("Ok, I added %s to the new %s list", item, args[1])
					}
				}
			} else {
				r.Reply("Sorry, I didn't get an answer I understand")
				return
			}
		} else {
			citem := strings.ToLower(item)
			for _, li := range list {
				if citem == strings.ToLower(li) {
					r.Say("%s is already on the %s list", item, args[1])
					return
				}
			}
			list = append(list, item)
			lists[listName] = list
			mret := r.UpdateDatum(datumKey, lock, lists)
			if mret != robot.Ok {
				r.Log(robot.Error, "Couldn't update lists: %s", mret)
				r.Reply("Crud. I had a problem saving my lists - somebody better check the log")
			} else {
				updated = true
			}
			r.Say("Ok, I added %s to the %s list", item, args[1])
		}
	}
	return
}

func init() {
	bot.RegisterPlugin("lists", robot.PluginHandler{
		Handler: lists,
		Config:  &config{},
	})
}
