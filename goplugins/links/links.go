// Package links implements a links demonstrator plugin showing how you
// can use the robot's brain to remember things - like links of items.
package links

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
)

const datumNameDefault = "links"
const maxLinkLen = 7

var spaces = regexp.MustCompile(`\s+`)

type config struct {
	Scope string
}

// Define the handler function
func links(r *bot.Robot, command string, args ...string) (retval bot.PlugRetVal) {
	if command == "init" { // ignore init
		return
	}
	// Create an empty map to unmarshal into
	links := make(map[string]string)
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
			if !updated {
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
	case "find":
		find := strings.ToLower(spaces.ReplaceAllString(args[0], ` `))
		linkList := make([]string, 0, 5)
		linkList = append(linkList, fmt.Sprintf("Here's what I have for \"%s\":", args[0]))
		var last string
		for link, lookup := range links {
			if strings.Contains(strings.ToLower(lookup), find) {
				linkList = append(linkList, link+": "+lookup)
				last = link
			}
		}
		if len(linkList) > 1 {
			r.Say(strings.Join(linkList, "\n"))
			if len(linkList) == 2 {
				r.RememberContext("link", last)
			}
		} else {
			r.Say(fmt.Sprintf("Sorry, I don't have any links for \"%s\"", args[0]))
		}
	case "list":
		linkslist := make([]string, 0, 7)
		linkslist = append(linkslist, "Here are the links I know about:")
		for link, lookup := range links {
			linkslist = append(linkslist, link+": "+lookup)
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
		var link, lookup string
		var prompted, replace bool
		if command == "add" {
			link = args[1]
			lookup = spaces.ReplaceAllString(args[0], ` `)
		} else {
			link = args[0]
		}
		current, exists := links[link]
		if exists {
			prompted = true
			r.CheckinDatum(datumKey, lock)
			r.Say(fmt.Sprintf("I already have that link associated with: %s", current))
			rep, ret := r.PromptForReply("YesNo", "Do you want me to replace it?")
			if ret == bot.Ok {
				switch strings.ToLower(rep) {
				case "n", "no":
					r.Say("Ok, I'll keep the old one")
					return
				default:
					r.Say("Ok, I'll replace the old one")
					replace = true
				}
			} else {
				r.Reply("Sorry, I didn't get an answer I understand")
				return
			}
		}
		if len(lookup) == 0 {
			prompted = true
			r.CheckinDatum(datumKey, lock)
			prompt := "What keywords or phrase do you want to attach to the link?"
			rep, ret := r.PromptForReply("lookup", prompt)
			if ret == bot.Ok {
				lookup = spaces.ReplaceAllString(rep, ` `)
			} else {
				r.Reply("Sorry, I didn't get your keywords / phrase")
				return
			}
		}
		if prompted {
			lock, _, _ = r.CheckoutDatum(datumKey, &links, true)
		}
		if _, exists := links[link]; exists && !replace {
			r.Reply("Incredible - somebody JUST saved that link! You'll have to try again.")
			return
		}
		links[link] = lookup
		mret := r.UpdateDatum(datumKey, lock, links)
		if mret != bot.Ok {
			r.Log(bot.Error, "Couldn't update links", mret)
			r.Reply("Crud. I had a problem saving the links - you can try again or ask an administrator to check the log")
			return
		}
		r.Say("Link added")
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
