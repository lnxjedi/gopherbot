package bot

import (
	"fmt"
	"strings"
)

/* builtin plugins, like help */

var builtIns []Plugin = []Plugin{
	{
		Name: "builtInhelp", // MUST match registered name below
		CommandMatches: []InputMatcher{
			InputMatcher{
				Regex:   `help ?([\d\w]+)?`,
				Command: "help",
			},
		},
		MessageMatches: []InputMatcher{
			InputMatcher{
				Regex:   `^(?i)help$`,
				Command: "barehelp",
			},
		},
	},
	{
		Name: "builtInreload", // MUST match registered name below
		CommandMatches: []InputMatcher{
			InputMatcher{
				Regex:   `reload`,
				Command: "reload",
			},
		},
	},
}

func help(bot Robot, channel, user, command string, args ...string) {
	// Get access to the underlying struct
	b := bot.robot
	if command == "barehelp" { // user just typed 'help' - the robot should introduce itself
		b.lock.RLock()
		reply := "Hello, I'm "
		if len(b.fullName) > 0 {
			reply += b.fullName + ", but you should just call me " + b.name + ".\n"
		} else {
			reply += b.name + ".\n"
		}
		reply += "I'm one of the staff robots available to your team, and can perform a variety of tasks. To find out what I can do, try:\n"
		reply += b.name + ", help (keyword)\nor:\nhelp (keyword), " + b.name + "\n"
		if b.alias != 0 {
			reply += "To save a little typing, you can prefix your message with my alias (" + string(b.alias) + "), like this:\n" + string(b.alias) + "help (keyword)\n"
		}
		reply += "The (keyword) is optional. If not supplied, you will get help for every command available to you in the current channel. If supplied, you will get help for every command with a matching keyword, along with the channels where it can be used. In all cases help will be sent as a direct message so the channels don't fill up with help text.\n"
		reply += "Additionally, some messages (like a bare 'help') will trigger commands as well, and help may or may not be available for those.\n\nFinally, if there's anything else you'd like to see me do, please contact my administrator"
		if len(b.adminContact) > 0 {
			reply += ", " + b.adminContact + "."
		} else {
			reply += "."
		}
		b.lock.RUnlock()
		bot.SendUserMessage(user, reply)
	}
	if command == "help" {
		b.lock.RLock()
		defer b.lock.RUnlock()

		var term, helpOutput string
		hasTerm := false
		if len(args) == 1 && len(args[0]) > 0 {
			hasTerm = true
			term = args[0]
			b.Log(Trace, "Help requested for term", term)
		}

		for _, plugin := range b.plugins {
			b.Log(Trace, fmt.Sprintf("Checking help for plugin %s (term: %s)", plugin.Name, term))
			if !hasTerm { // if you ask for help without a term, you just get help for whatever commands are available to you
				if b.messageAppliesToPlugin(user, channel, command, plugin) {
					for _, phelp := range plugin.Help {
						for _, helptext := range phelp.Helptext {
							helpOutput += helptext + string('\n')
						}
					}
				}
			} else { // when there's a search term, give all help for that term, but add (channels: xxx) at the end
				for _, phelp := range plugin.Help {
					for _, keyword := range phelp.Keywords {
						if term == keyword {
							chantext := ""
							for _, pchan := range plugin.Channels {
								if channel != pchan {
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
								helpOutput += helptext + string('\n')
							}
						}
					}
				}
			}
		}
		if len(helpOutput) == 0 {
			bot.SendUserMessage(user, "Sorry, bub - I got nothin' for ya'")
		} else {
			bot.SendUserMessage(user, strings.TrimRight(helpOutput, "\n"))
		}
	}
}

func reload(bot Robot, channel, user, command string, args ...string) {
	// Get access to the underlying struct
	b := bot.robot
	if command == "reload" {
		if b.CheckAdmin(user) {
			err := b.loadConfig()
			if err != nil {
				bot.Reply("Error encountered during reload, check the logs")
				b.Log(Error, fmt.Errorf("Reloading configuration, requested by %s: %v", user, err))
				return
			}
			bot.Reply("Configuration reloaded successfully")
			b.Log(Info, "Configuration successfully reloaded after a request from:", user)
		} else {
			bot.Reply("Sorry, only an admin user can request that")
		}
	}
}

func init() {
	RegisterPlugin("builtInhelp", help)     // MUST match plugin name above
	RegisterPlugin("builtInreload", reload) // MUST match plugin name above
}
