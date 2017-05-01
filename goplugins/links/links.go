// Package links implements a links demonstrator plugin showing how you
// can use the robot's brain to remember things - like links of items.
package links

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/uva-its/gopherbot/bot"
)

const datumName = "links"

var spaces = regexp.MustCompile(`\s+`)

type config struct {
	Scope string
}

// Define the handler function
func links(r *bot.Robot, command string, args ...string) {
	// Create an empty map to unmarshal into
	if command == "init" { // ignore init
		return
	}
	var links = make(map[string][]string)
	var lock string
	scope := &config{}

	datumKey := datumName // default global
	ret := r.GetPluginConfig(&scope)
	if ret == bot.Ok {
		if strings.ToLower(scope.Scope) == "channel" {
			datumKey = r.Channel + ":" + datumName
		}
	}

	updated := false
	switch command {
	case "help":
		r.Say(longHelp)
		return
	case "find": // read-only case(s)
		_, _, ret = r.CheckoutDatum(datumKey, &links, false)
	default:
		// all other cases are read-write
		lock, _, ret = r.CheckoutDatum(datumKey, &links, true)
		defer func() {
			if updated {
				mret := r.UpdateDatum(datumName, lock, links)
				if mret != bot.Ok {
					r.Log(bot.Error, "Couldn't update links", mret)
					r.Reply("Crud. I had a problem saving the links - you can try again or ask an administrator to check the log")
				}
			} else {
				// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
				// in case there's another thread waiting.
				r.CheckinDatum(datumName, lock)
			}
		}()
	}
	if ret != bot.Ok {
		r.Log(bot.Error, fmt.Sprintf("Couldn't load links: %s", ret))
		r.Reply("I had a problem loading the links, somebody should check my log file")
		r.CheckinDatum(datumName, lock) // well-behaved plugins using the brain will always check in data when done
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
	case "add", "save":
		lookups := make([]string, 0, 5)
		var link string
		if command == "add" {
			link = args[1]
			lookups = []string{
				spaces.ReplaceAllString(args[0], ` `),
			}
		} else {
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
		}
		if len(lookups) > 0 {
			links[link] = lookups
			r.Say("Ok, link added")
			updated = true
		} else {
			r.Say("Not adding link with no lookups")
		}
	case "find":
		lookup := strings.ToLower(spaces.ReplaceAllString(args[0], ` `))
		linkList := make([]string, 0, 5)
		linkList = append(linkList, fmt.Sprintf("Here's what I have for \"%s\":", args[0]))
		for link, lookups := range links {
		loop:
			for _, key := range lookups {
				if strings.Contains(strings.ToLower(key), lookup) {
					linkList = append(linkList, key+": "+link)
					break loop
				}
			}
		}
		if len(linkList) > 1 {
			r.Say(strings.Join(linkList, "\n"))
			if len(linkList) == 1 {
				r.RememberContext("link", linkList[0])
			}
		} else {
			r.Say(fmt.Sprintf("Sorry, I don't have any links for \"%s\"", lookup))
		}
	}
}

func init() {
	bot.RegisterPlugin("links", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       links,
	})
}
