// Package links implements a links demonstrator plugin showing how you
// can use the robot's brain to remember things - like links of items.
package links

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/uva-its/gopherbot/bot"
)

const maxlinks = 14
const maxitems = 42
const maxitemlen = 42
const maxlinknamelen = 21
const datumName = "linkmap"

type itemlink []string

// Define the handler function
func links(r *bot.Robot, command string, args ...string) {
	// Create an empty map to unmarshal into
	if command == "init" { // ignore init
		return
	}
	var links = make(map[string]itemlink)
	var lock string
	var ret bot.RetVal
	// First, check out the link
	switch command {
	case "show", "send":
		// read-only cases
		_, _, ret = r.CheckoutDatum(datumName, &links, false)
	default:
		// all other cases are read-write
		lock, _, ret = r.CheckoutDatum(datumName, &links, true)
	}
	if ret != bot.Ok {
		r.Log(bot.Error, fmt.Sprintf("Couldn't load links: %s", ret))
		r.Reply("I had a problem loading the links, somebody should check my log file")
		r.CheckinDatum(datumName, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}
	updated := false
	defer func() {
		if updated {
			ret := r.UpdateDatum(datumName, lock, links)
			if ret != bot.Ok {
				r.Log(bot.Error, "Coudln't update links")
				r.Reply("Crud. I had a problem saving my links - somebody better check the log")
			}
		} else {
			// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
			// in case there's another thread waiting.
			r.CheckinDatum(datumName, lock)
		}
	}()
	switch command {
	case "remove":
		item := args[0]
		linkName := strings.Replace(strings.ToLower(args[1]), " ", "-", -1)
		linkKey := r.Channel + "~" + linkName
		link, ok := links[linkKey]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a link named %s", linkName))
			return
		}
		citem := strings.ToLower(item)
		found := false
		for i, li := range link {
			if citem == strings.ToLower(li) {
				link[i] = link[len(link)-1]
				link = link[:len(link)-1]
				links[linkKey] = link
				r.Say(fmt.Sprintf("Ok, I removed %s from the %s link", item, linkName))
				updated = true
				found = true
			}
		}
		if !found {
			r.Say(fmt.Sprintf("I didn't see %s on the %s link", item, linkName))
			return
		}
	case "empty", "delete":
		linkName := strings.Replace(strings.ToLower(args[1]), " ", "-", -1)
		linkKey := r.Channel + "~" + linkName
		_, ok := links[linkKey]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a link named %s", linkName))
			return
		}
		updated = true
		if command == "empty" {
			links[linkKey] = []string{}
			r.Say("Emptied")
		} else {
			delete(links, linkName)
			r.Say("Deleted")
		}
	case "link":
		var found int
		var linksBuffer bytes.Buffer
		for l := range links {
			lchan := strings.Split(l, "~")[0]
			if lchan == r.Channel {
				found++
				fmt.Fprintf(&linksBuffer, "%s\n", strings.Split(l, "~")[1])
			}
		}
		if found == 0 {
			r.Say("I don't have any links for this channel")
			return
		}
		r.Say(fmt.Sprintf("Here are the links I have for this channel:\n%s", linksBuffer.String()))
	case "show", "send":
		linkName := strings.Replace(strings.ToLower(args[1]), " ", "-", -1)
		linkKey := r.Channel + "~" + linkName
		var linkBuffer bytes.Buffer
		link, ok := links[linkKey]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a link named %s", linkName))
			return
		}
		if len(link) == 0 {
			r.Say(fmt.Sprintf("The %s link is empty", linkName))
			return
		}
		lineEnd := "\n"
		if command == "send" {
			lineEnd = "\r\n"
		}
		for _, li := range link {
			fmt.Fprintf(&linkBuffer, "%s%s", li, lineEnd)
		}
		switch command {
		case "show":
			r.Say(fmt.Sprintf("Here's what I have on the %s link:\n%s", linkName, linkBuffer.String()))
		case "send":
			if ret := r.Email(fmt.Sprintf("The %s link", linkName), &linkBuffer); ret != bot.Ok {
				r.Say("Sorry, there was an error sending the email - have somebody check the my log file")
				return
			}
			botmail := r.GetBotAttribute("email").String()
			r.Say(fmt.Sprintf("Ok, I sent the %s link to you - look for email from %s", linkName, botmail))
		}
	case "pick":
		linkName := strings.Replace(strings.ToLower(args[1]), " ", "-", -1)
		linkKey := r.Channel + "~" + linkName
		link, ok := links[linkKey]
		if !ok {
			r.Say(fmt.Sprintf("I don't have a link named %s", linkName))
			return
		}
		if len(link) == 0 {
			r.Say(fmt.Sprintf("The %s link is empty", linkName))
			return
		}
		item := r.RandomString(link)
		r.Say(fmt.Sprintf("Here you go: %s", item))
	case "add":
		// Case sensitive input, case insensitve equality checking
		item := args[0]
		linkName := strings.Replace(strings.ToLower(args[1]), " ", "-", -1)
		linkKey := r.Channel + "~" + linkName
		if len(item) > maxitemlen {
			r.Say(fmt.Sprintf("Sorry, that item is too big - the longest I'll take is %d", maxitemlen))
			return
		}
		if len(linkName) > maxlinknamelen {
			r.Say(fmt.Sprintf("Sorry, that link name is too big - the longest I'll take is %d", maxlinknamelen))
			return
		}
		link, ok := links[linkKey]
		if !ok {
			if len(links) >= maxlinks {
				r.Say(fmt.Sprintf("Sorry, can't create \"%s\", there are too many links already", linkName))
				return
			}
			links[linkKey] = []string{item}
		} else {
			citem := strings.ToLower(item)
			for _, li := range link {
				if citem == strings.ToLower(li) {
					r.Say(fmt.Sprintf("%s is already on the %s link", item, linkName))
					return
				}
			}
			link = append(link, item)
			links[linkKey] = link
		}
		r.Say(fmt.Sprintf("Ok, I added %s to the %s link", item, linkName))
		updated = true
	}
}

func init() {
	bot.RegisterPlugin("links", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       links,
	})
}
