package bot

import (
	"fmt"
	"strings"
)

/* builtin plugins, like help */

var builtIns []Plugin = []Plugin{
	{
		Name:        "builtInhelp", // MUST match registered name below
		AllowDirect: true,
		CommandMatches: []InputMatcher{
			InputMatcher{
				Regex:   `help ?([\d\w]+)?`,
				Command: "help",
			},
		},
	},
	{
		Name:        "builtInreload", // MUST match registered name below
		AllowDirect: true,
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
			if !hasTerm { // if you ask for help without a term, you just get whatever commands are available in your channel
				if len(plugin.Channels) > 0 {
					b.Log(Trace, fmt.Sprintf("Seeing if %s is available in channel %s", plugin.Name, channel))
					for _, pchan := range plugin.Channels {
						b.Log(Trace, fmt.Sprintf("... checking current channel %s against plugin channel %s", channel, pchan))
						if channel == pchan {
							for _, phelp := range plugin.Help {
								for _, helptext := range phelp.Helptext {
									helpOutput += helptext + string('\n')
								}
							}
						}
					}
				} else {
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
			bot.Reply("Sorry, bub - I got nothin' for ya'")
		} else {
			bot.Say(strings.TrimRight(helpOutput, "\n"))
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
