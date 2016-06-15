// The lists plugin is a small demonstrator plugin showing how you
// can use the robot's brain to remember things - like lists of items.
package lists

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/parsley42/gopherbot/bot"
)

const maxlists = 7
const maxitems = 28
const maxitemlen = 21
const maxlistnamelen = 14
const datumName = "listmap"

type itemList []string

// Define the handler function
func lists(r bot.Robot, command string, args ...string) {
	// Create an empty map to unmarshal into
	if command == "init" { // ignore init
		return
	}
	var lists = make(map[string]itemList)
	var lock string
	var ret bot.BotRetVal
	// First, check out the list
	switch command {
	case "show", "send":
		// read-only cases
		_, _, ret = r.CheckoutDatum(datumName, &lists, false)
	default:
		// all other cases are read-write
		lock, _, ret = r.CheckoutDatum(datumName, &lists, true)
	}
	if ret != bot.Ok {
		r.Log(bot.Error, "Couldn't load lists")
		r.Reply("I had a problem loading the lists, somebody should check my log file")
		r.Checkin(datumName, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}
	updated := false
	defer func() {
		if updated {
			ret := r.UpdateDatum(datumName, lock, lists)
			if ret != bot.Ok {
				r.Log(bot.Error, "Coudln't update lists")
				r.Reply("Crud. I had a problem saving my lists - somebody better check the log")
			}
		} else {
			// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
			// in case there's another thread waiting.
			r.Checkin(datumName, lock)
		}
	}()
	switch command {
	case "remove":
		item := args[0]
		listName := strings.ToLower(args[1])
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
		listName := args[0]
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
		if len(lists) == 0 {
			r.Say("I don't have any lists")
			return
		}
		var listsBuffer bytes.Buffer
		for l, _ := range lists {
			fmt.Fprintf(&listsBuffer, "%s\n", l)
		}
		r.Say(fmt.Sprintf("Here are the lists I have:\n%s", listsBuffer.String()))
	case "show", "send":
		listName := args[0]
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
			botmail, _ := r.GetBotAttribute("email")
			r.Say(fmt.Sprintf("Ok, I sent the %s list to you - look for email from %s", listName, botmail))
		}
	case "add":
		// Case sensitive input, case insensitve equality checking
		item := args[0]
		listName := strings.ToLower(args[1])
		if len(item) > maxitemlen {
			r.Say(fmt.Sprintf("Sorry, that item is too big - the longest I'll take is %d", maxitemlen))
			return
		}
		if len(listName) > maxlistnamelen {
			r.Say(fmt.Sprintf("Sorry, that list name is too big - the longest I'll take is %d", maxlistnamelen))
			return
		}
		list, ok := lists[listName]
		if !ok {
			if len(lists) >= maxlists {
				r.Say(fmt.Sprintf("Sorry, can't create \"%s\", there are too many lists already", listName))
				return
			}
			lists[listName] = []string{item}
		} else {
			citem := strings.ToLower(item)
			for _, li := range list {
				if citem == strings.ToLower(li) {
					r.Say(fmt.Sprintf("%s is already on the %s list", item, listName))
					return
				}
			}
			list := append(list, item)
			lists[listName] = list
		}
		r.Say(fmt.Sprintf("Ok, I added %s to the %s list", item, listName))
		updated = true
	}
}

func init() {
	bot.RegisterPlugin("lists", bot.PluginHandler{Handler: lists})
}
