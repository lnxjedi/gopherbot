// Package links implements a links demonstrator plugin showing how you
// can use the robot's brain to remember things - like links of items.
package links

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/uva-its/gopherbot/bot"
)

const datumNameDefault = "links"
const maxLinkLen = 7

var spaces = regexp.MustCompile(`\s+`)

type config struct {
	Scope string
}

// Define the handler function
func links(r *bot.Robot, command string, args ...string) (retval bot.PlugRetVal) {
	// Create an empty map to unmarshal into
	if command == "init" { // ignore init
		return
	}
	links := make(map[string][]string)
	var lock string
	scope := &config{}

	datumKey := datumNameDefault // default global
	ret := r.GetPluginConfig(&scope)
	if ret == bot.Ok {
		if strings.ToLower(scope.Scope) == "channel" {
			datumKey = r.Channel + ":" + datumNameDefault
		}
	}

	updated := false
	switch command {
	case "help":
		r.Say(longHelp)
		return
	case "find", "list": // read-only case(s)
		_, _, ret = r.CheckoutDatum(datumKey, &links, false)
	default:
		// all other cases are read-write
		lock, _, ret = r.CheckoutDatum(datumKey, &links, true)
		defer func() {
			if updated {
				mret := r.UpdateDatum(datumKey, lock, links)
				if mret != bot.Ok {
					r.Log(bot.Error, "Couldn't update links", mret)
					r.Reply("Crud. I had a problem saving the links - you can try again or ask an administrator to check the log")
				}
			} else {
				// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
				// in case there's another thread waiting.
				r.CheckinDatum(datumKey, lock)
			}
		}()
	}
	if ret != bot.Ok {
		r.Log(bot.Error, fmt.Sprintf("Couldn't load links: %s", ret))
		r.Reply("I had a problem loading the links, somebody should check my log file")
		r.CheckinDatum(datumKey, lock) // well-behaved plugins using the brain will always check in data when done
		return
	}
	switch command {
	case "remove":
		link := args[0]
		_, ok := links[link]
		if !ok {
			r.Say(fmt.Sprintf("I don't have the link %s", link))
			return
		}
		delete(links, link)
		r.Say(fmt.Sprintf("Ok, I removed the link %s", link))
		updated = true
	case "list":
		linkslist := make([]string, 0, 7)
		linkslist = append(linkslist, "Here are the links I know about:")
		for link, keys := range links {
			linkslist = append(linkslist, link+": "+strings.Join(keys, ", "))
		}
		if len(linkslist) > 1 {
			if len(linkslist) > maxLinkLen {
				r.Say("I know a LOT of links - so I sent you a direct message")
				r.Direct().Say(strings.Join(linkslist, "\n"))
			} else {
				r.Say(strings.Join(linkslist, "\n"))
			}
		} else {
			r.Say("I haven't stored any links yet")
		}
	case "add", "save":
		lookups := make([]string, 0, 5)
		var link string
		if command == "add" {
			link = args[1]
			lookups = []string{
				spaces.ReplaceAllString(args[0], ` `),
			}
		} else {
			r.CheckinDatum(datumKey, lock)
			link = args[0]
			r.Say("Ok, what keywords or phrases do you want to attach to the link? (\"done\" when finished)")
			for {
				rep, ret := r.WaitForReply("lookup", 60)
				if ret == bot.Ok {
					if strings.ToLower(rep) == "done" {
						break
					}
					lookup := spaces.ReplaceAllString(rep, ` `)
					lookups = append(lookups, lookup)
					r.Say(fmt.Sprintf("Added \"%s\", type \"done\" if you're finished", lookup))
				} else {
					break
				}
			}
			if len(lookups) > 0 {
				lock, _, _ = r.CheckoutDatum(datumKey, &links, true)
			}
		}
		if len(lookups) > 0 {
			keys, exists := links[link]
			if exists {
				for _, i := range lookups {
					newkey := true
					lk := strings.ToLower(i)
					for _, j := range keys {
						if lk == strings.ToLower(j) {
							newkey = false
							break
						}
					}
					if newkey {
						updated = true
						keys = append(keys, i)
					}
				}
				if updated {
					links[link] = keys
					r.Say("Ok, updated keys for existing link")
				} else {
					r.Say("No new keys given for existing link")
				}
			} else {
				links[link] = lookups
				r.Say("Ok, link added")
				updated = true
			}
		} else {
			r.Say("Not adding link with no lookups")
		}
	case "find":
		lookup := strings.ToLower(spaces.ReplaceAllString(args[0], ` `))
		linkList := make([]string, 0, 5)
		linkList = append(linkList, fmt.Sprintf("Here's what I have for \"%s\":", args[0]))
		var last string
		for link, lookups := range links {
		loop:
			for _, key := range lookups {
				if strings.Contains(strings.ToLower(key), lookup) {
					linkList = append(linkList, key+": "+link)
					last = link
					break loop
				}
			}
		}
		if len(linkList) > 1 {
			r.Say(strings.Join(linkList, "\n"))
			if len(linkList) == 2 {
				r.RememberContext("link", last)
			}
		} else {
			r.Say(fmt.Sprintf("Sorry, I don't have any links for \"%s\"", lookup))
		}
	}
	return
}

func init() {
	bot.RegisterPlugin("links", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       links,
		Config:        &config{},
	})
}
