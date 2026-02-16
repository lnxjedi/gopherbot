// Package help - plugin spits out a helpful message when a user just types "help" in
// a channel, and also responds when the user addresses the robot but no plugin
// matched. Advanced users will probably disable this one and write their own.
package help

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// Define the handler function
func help(bot robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	botName := bot.GetBotAttribute("name").String()
	if command == "help" { // user just typed 'help' - the robot should introduce itself
		botContact := bot.GetBotAttribute("contact").String()
		botAlias := bot.GetBotAttribute("alias").String()
		reply := "Hi, I'm "
		reply += strings.Title(botName) + ", a staff robot. I see you've asked for help.\n\n"
		reply += "I've been programmed to perform a variety of tasks for your team, and I'll respond when you send me commands that match specific patterns. " +
			"You can ask for command help by addressing me with my name, for example:\n\n"
		reply += botName + ", help ping\nor:\nhelp ping, " + botName + "\n\n"
		if len(botAlias) > 0 {
			reply += "To save a little typing, you can also use my alias ( " + botAlias + " ). For example:\n\n" + botAlias + "help ping\n\n"
		}
		reply += "Here are the most useful discovery commands:\n\n"
		reply += "• " + botAlias + "help - quick help and pointers\n"
		reply += "• " + botAlias + "commands - browse command groups available in this channel\n"
		reply += "• " + botAlias + "help <keyword> - show the best matches with usage and examples\n"
		reply += "• " + botAlias + "help-all - detailed help for commands available here\n\n"
		reply += "If I can't match a command, I'll usually reply with the closest matches and a suggested next help command so you can recover quickly.\n\n"
		reply += "When command help shows syntax, (something) means optional and <something> means required. Command availability can vary by channel, direct message, and permissions, so help output may differ from place to place.\n\n"
		reply += "Also, from time to I may ask you a question, prompting for additional information - these messages will mention you by name if not in a private conversation. You only need to type your reply - if you address me by name (or alias), I'll consider it a new command and send an error to the plugin requesting input. Additionally, there are two special replies I understand: \"=\" means 'use the default value', whatever that might be; \"-\" means 'cancel', returning an error value to the plugin.\n\n"
		if bot.GetMessage().Protocol.String() == "Terminal" {
			reply += "Since you're using the 'terminal' connector, you can get simple help for changing the user, channel and thread just by hitting <return> with an empty command.\n\n"
		}
		reply += "For basic information about me, you can use my \"info\" command. Finally, if there's anything else you'd like to see me do, please contact my administrator"
		if len(botContact) > 0 {
			reply += ", " + botContact + "."
		} else {
			reply += "."
		}
		bot.SayThread(reply)
	}
	return
}

func init() {
	robot.RegisterPlugin("help", robot.PluginHandler{Handler: help})
}
