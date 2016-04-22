package bot

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

func help(bot Robot, channel, user, command string, args ...string) error {
	b := bot.Gobot
	if command == "help" {
		b.Log(Debug, "Sombebody asked for help")
	}
	return nil
}

func reload(bot Robot, channel, user, command string, args ...string) error {
	b := bot.Gobot
	if command == "reload" {
		b.Log(Debug, "Somebody requested a reload")
	}
	return nil
}

func init() {
	RegisterPlugin("builtInhelp", help)     // MUST match plugin name above
	RegisterPlugin("builtInreload", reload) // MUST match plugin name above
}
