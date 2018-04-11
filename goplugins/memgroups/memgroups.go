// Package lists implements a lists demonstrator plugin showing how you
// can use the robot's brain to remember things - like lists of items.
package lists

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
)

const datumName = "listmap"

var spaces = regexp.MustCompile(`\s+`)

type itemList []string

type config struct {
	Scope string
}

func name(arg string) string {
	return strings.ToLower(spaces.ReplaceAllString(arg, " "))
}

// Define the handler function
func lists(r *bot.Robot, command string, args ...string) (retval bot.PlugRetVal) {
	// Create an empty map to unmarshal into
	if command == "init" { // ignore init
		return
	}
	var lists = make(map[string]itemList)
	var lock string
	var ret bot.RetVal
	scope := &config{}

	datumKey := datumName // default global
	ret = r.GetPluginConfig(&scope)
	r.Log(bot.Debug, fmt.Sprintf("Retrieved lists config: %v", scope))
	if ret == bot.Ok {
		if strings.ToLower(scope.Scope) == "channel" {
			datumKey = r.Channel + ":" + datumName
		}
	}

	updated := false
	// First, check out the list
	switch command {
	case "help":
		r.Say(listHelp)
		return
	case "show", "send", "pick", "list":
		// read-only cases
		_, _, ret = r.CheckoutDatum(datumKey, &lists, false)
	default:
		// all other cases are read-write
		lock, _, ret = r.CheckoutDatum(datumKey, &lists, true)
		defer func() {
			if updated {
				mret := r.UpdateDatum(datumKey, lock, lists)
				if mret != bot.Ok {
					r.Log(bot.Error, "Coudln't update lists")
					r.Reply("Crud. I had a problem saving my lists - somebody better check the log")
				}
			} else {
				// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
				// in case there's another thread waiting.
				r.CheckinDatum(datumKey, lock)
			}
		}()
	}
	if ret != bot.Ok {
		r.Log(bot.Error, fmt.Sprintf("Couldn't load lists: %s", ret))
		r.Reply("I had a problem loading the lists, somebody should check my log file")
		r.CheckinDatum(datumKey, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}
	switch command {
	case "remove":
		item := args[0]
		listName := name(args[1])
		list, ok := lists[listName]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a list named %s", listName))
			return
		}
		citem := strings.ToLower(item)
		found := false
		for i, li := range list {
			if citem == strings.ToLower(li) {
				list[i] = list[len(list)-1]
				list = list[:len(list)-1]
				lists[listName] = list
				r.Say(fmt.Sprintf("Ok, I removed %s from the %s list", item, listName))
				updated = true
				found = true
			}
		}
		if !found {
			r.Say(fmt.Sprintf("I didn't see %s on the %s list", item, listName))
			return
		}
	case "empty", "delete":
		listName := name(args[0])
		_, ok := lists[listName]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a list named %s", listName))
			return
		}
		updated = true
		if command == "empty" {
			lists[listName] = []string{}
			r.Say("Emptied")
		} else {
			delete(lists, listName)
			r.Say("Deleted")
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
			r.Say(fmt.Sprintf("Here are the lists I have for this channel:\n%s", strings.Join(listlist, "\n")))
		} else {
			r.Say(fmt.Sprintf("Here are the lists I know about:\n%s", strings.Join(listlist, "\n")))
		}
	case "show", "send":
		listName := name(args[0])
		var listBuffer bytes.Buffer
		list, ok := lists[listName]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a list named %s", listName))
			return
		}
		if len(list) == 0 {
			r.Say(fmt.Sprintf("The %s list is empty", listName))
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
			r.Say(fmt.Sprintf("Here's what I have on the %s list:\n%s", listName, listBuffer.String()))
		case "send":
			if ret := r.Email(fmt.Sprintf("The %s list", listName), &listBuffer); ret != bot.Ok {
				r.Say("Sorry, there was an error sending the email - have somebody check the my log file")
				return
			}
			botmail := r.GetBotAttribute("email").String()
			r.Say(fmt.Sprintf("Ok, I sent the %s list to you - look for email from %s", listName, botmail))
		}
	case "pick":
		listName := name(args[0])
		list, ok := lists[listName]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a list named %s", listName))
			return
		}
		if len(list) == 0 {
			r.Say(fmt.Sprintf("The %s list is empty", listName))
			return
		}
		item := r.RandomString(list)
		r.Say(fmt.Sprintf("Here you go: %s", item))
		r.RememberContext("item", item)
	case "add":
		// Case sensitive input, case insensitve equality checking
		item := args[0]
		listName := name(args[1])
		list, ok := lists[listName]
		if !ok {
			lists[listName] = []string{item}
		} else {
			citem := strings.ToLower(item)
			for _, li := range list {
				if citem == strings.ToLower(li) {
					r.Say(fmt.Sprintf("%s is already on the %s list", item, listName))
					return
				}
			}
			list = append(list, item)
			lists[listName] = list
		}
		r.Say(fmt.Sprintf("Ok, I added %s to the %s list", item, listName))
		updated = true
	}
	return
}

func init() {
	bot.RegisterPlugin("lists", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       lists,
		Config:        &config{},
	})
}
