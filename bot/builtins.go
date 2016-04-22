package bot

import "fmt"

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
		b.Log(Debug, "Sombebody asked for help")
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
