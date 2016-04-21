package bot

/* builtin plugins, like help */

var builtIns []Plugin = []Plugin{
	{
		Name:        "help",
		AllowDirect: true,
		CommandMatches: []InputMatcher{
			InputMatcher{
				Regex:   `help ?([\d\w]+)?`,
				Command: "help",
			},
		},
	},
	{
		Name:        "reload",
		AllowDirect: true,
		CommandMatches: []InputMatcher{
			InputMatcher{
				Regex:   `reload`,
				Command: "reload",
			},
		},
	},
}

var builtinHandlers map[string]func(bot Robot, channel, user, command string, args ...string) error = make(map[string]func(bot Robot, channel, user, command string, args ...string) error)

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
	builtinHandlers["help"] = help
	builtinHandlers["reload"] = reload
}
