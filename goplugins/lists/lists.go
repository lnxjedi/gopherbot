// ping implements the most trivial of Go plugins
package lists

import (
	"bytes"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/parsley42/gopherbot/bot"
)

const maxlists = 7
const maxitems = 28
const maxitemlen = 21
const maxlistnamelen = 14
const datumName = "listmap"

type itemList []string
type listMap map[string]itemList

// Define the handler function
func lists(r bot.Robot, command string, args ...string) {
	// Create an empty map to unmarshal into
	if command == "init" { // ignore init
		return
	}
	var lists = make(map[string]itemList)
	var lock string
	var err error
	// First, check out the list
	switch command {
	case "show", "send":
		// read-only cases
		_, _, err = r.CheckoutDatum(datumName, &lists, false)
	default:
		// all other cases are read-write
		lock, _, err = r.CheckoutDatum(datumName, &lists, true)
	}
	if err != nil {
		r.Log(bot.Error, fmt.Errorf("Loading list: %v", err))
		r.Reply("I had a problem loading the lists, somebody should check my log file")
		r.Checkin(datumName, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}
	updated := false
	defer func() {
		if updated {
			err := r.UpdateDatum(datumName, lock, lists)
			if err != nil {
				r.Log(bot.Error, fmt.Errorf("Updating list: %v", err))
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
		var mailTo, mailFrom, botName string
		if command == "send" {
			lineEnd = "\r\n"
			mailFrom = r.GetBotAttribute("email")
			botName = r.GetBotAttribute("fullName")
			mailTo = r.GetSenderAttribute("email")
			if len(mailFrom) == 0 {
				r.Say("Sorry, I don't have an email address to send from!")
				return
			}
			if len(mailTo) == 0 {
				r.Say("Sorry, I wasn't able to look up your email address")
				return
			}
			fmt.Fprintf(&listBuffer, "From: %s <%s>\r\n", botName, mailFrom)
			fmt.Fprintf(&listBuffer, "Subject: The %s list\r\n\r\n", listName)
		}
		for _, li := range list {
			fmt.Fprintf(&listBuffer, "%s%s", li, lineEnd)
		}
		switch command {
		case "show":
			r.Say(fmt.Sprintf("Here's what I have on the %s list:\n%s", listName, listBuffer.String()))
		case "send":
			// Connect to the remote SMTP server.
			c, err := smtp.Dial("127.0.0.1:25")
			if err != nil {
				r.Say("Couldn't connect to localhost email server, have somebody check my log file")
				r.Log(bot.Error, fmt.Errorf("Sending email: %v", err))
				return
			}
			if err := c.Mail(mailFrom); err != nil {
				r.Say("Sorry, I had a problem setting the sender, have somebody check my log file")
				r.Log(bot.Error, fmt.Errorf("Setting sender to %s: %v", mailFrom, err))
				return
			}
			if err := c.Rcpt(mailTo); err != nil {
				r.Say("Sorry, I had a problem setting the recipient, have somebody check my log file")
				r.Log(bot.Error, fmt.Errorf("Setting recipient to %s: %v", mailTo, err))
				return
			}
			// Send the email body.
			wc, err := c.Data()
			if err != nil {
				r.Say("Sorry, I had a problem starting the message body, have somebody check my log file")
				r.Log(bot.Error, fmt.Errorf("Starting message body: %v", err))
				return
			}
			_, err = wc.Write(listBuffer.Bytes())
			if err != nil {
				r.Say("Sorry, I had a problem sending the message body, have somebody check my log file")
				r.Log(bot.Error, fmt.Errorf("Sending message body: %v", err))
				return
			}
			err = wc.Close()
			if err != nil {
				r.Say("Sorry, I had a problem closing the message body, have somebody check my log file")
				r.Log(bot.Error, fmt.Errorf("Closing message body: %v", err))
				return
			}
			err = c.Quit()
			if err != nil {
				r.Say("Sorry, I had a problem closing the mail connection, have somebody check my log file")
				r.Log(bot.Error, fmt.Errorf("Closing mail connection: %v", err))
				return
			}
			r.Say(fmt.Sprintf("Ok, I sent the %s list to %s - look for an email from %s", listName, mailTo, mailFrom))
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
	bot.RegisterPlugin("lists", lists)
}
