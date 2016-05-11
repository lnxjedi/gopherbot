package bot

import (
	"fmt"
	"strings"
)

// if help is more than tooLong lines long, send a private message
const tooLong = 7

// If this list doesn't match what's registered below,
// you're gonna have a bad time
var builtIns = []string{
	"builtInhelp",
	"builtInreload",
	"builtIndump",
}

func init() {
	RegisterPlugin("builtIndump", dump)
	RegisterPlugin("builtInhelp", help)
	RegisterPlugin("builtInreload", reload)
}

/* builtin plugins, like help */

func help(bot Robot, command string, args ...string) {
	// Get access to the underlying struct
	b := bot.robot
	if command == "help" {
		b.lock.RLock()
		defer b.lock.RUnlock()

		var term, helpOutput string
		hasTerm := false
		helpLines := 0
		if len(args) == 1 && len(args[0]) > 0 {
			hasTerm = true
			term = args[0]
			b.Log(Trace, "Help requested for term", term)
		}

		for _, plugin := range b.plugins {
			b.Log(Trace, fmt.Sprintf("Checking help for plugin %s (term: %s)", plugin.Name, term))
			if !hasTerm { // if you ask for help without a term, you just get help for whatever commands are available to you
				if b.messageAppliesToPlugin(bot.User, bot.Channel, command, plugin) {
					for _, phelp := range plugin.Help {
						for _, helptext := range phelp.Helptext {
							helpOutput += helptext + string('\n')
							helpLines++
						}
					}
				}
			} else { // when there's a search term, give all help for that term, but add (channels: xxx) at the end
				for _, phelp := range plugin.Help {
					for _, keyword := range phelp.Keywords {
						if term == keyword {
							chantext := ""
							for _, pchan := range plugin.Channels {
								if bot.Channel != pchan {
									if len(chantext) == 0 {
										chantext += " (channels: " + pchan
									} else {
										chantext += ", " + pchan
									}
								}
							}
							if len(chantext) != 0 {
								chantext += ")"
							}
							for _, helptext := range phelp.Helptext {
								helpOutput += helptext + chantext + string('\n')
								helpLines++
							}
						}
					}
				}
			}
		}
		switch {
		case helpLines == 0:
			bot.Say("Sorry, bub - I got nothin' for ya'")
		case helpLines > tooLong:
			if len(bot.Channel) > 0 {
				bot.Reply("(the help for this channel was pretty long, so I sent you a private message)")
				helpOutput = "Help for channel: " + bot.Channel + "\n" + helpOutput
			}
			bot.SendUserMessage(bot.User, strings.TrimRight(helpOutput, "\n"))
		default:
			bot.Say(strings.TrimRight(helpOutput, "\n"))
		}
	}
}

func dump(bot Robot, command string, args ...string) {
	// Get access to the underlying struct
	b := bot.robot
	if !bot.CheckAdmin() {
		bot.Reply("Sorry, only an admin user can request that")
		return
	}
	switch command {
	case "robot":
		bot.Fixed().Say(fmt.Sprintf("%+v", bot))
	case "plugin":
		b.lock.RLock()
		defer b.lock.RUnlock()
		found := false
		for _, plugin := range b.plugins {
			if args[0] == plugin.Name {
				found = true
				bot.Fixed().Say(fmt.Sprintf("%+v", plugin))
				bot.Log(Info, fmt.Sprintf("Dump of plugin %s:\n%+v", args[0], plugin))
			}
		}
		if !found {
			bot.Say("Didn't find a plugin named " + args[0])
		}
	}
}

func reload(bot Robot, command string, args ...string) {
	// Get access to the underlying struct
	b := bot.robot
	if command == "reload" {
		if bot.CheckAdmin() {
			err := b.loadConfig()
			if err != nil {
				bot.Reply("Error encountered during reload, check the logs")
				b.Log(Error, fmt.Errorf("Reloading configuration, requested by %s: %v", bot.User, err))
				return
			}
			bot.Reply("Configuration reloaded successfully")
			b.Log(Info, "Configuration successfully reloaded after a request from:", bot.User)
		} else {
			bot.Reply("Sorry, only an admin user can request that")
		}
	}
}
