// Package help - plugin spits out a helpful message when a user just types "help" in
// a channel, and also responds when the user addresses the robot but no plugin
// matched. Advanced users will probably disable this one and write their own.
package help

import (
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/lnxjedi/gopherbot/robot"
)

var (
	gobot   bot.Robot
	botName string
)

// Define the handler function
func help(bot robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	m := bot.GetMessage()
	botName := bot.GetBotAttribute("name").String()
	if command == "help" { // user just typed 'help' - the robot should introduce itself
		botContact := bot.GetBotAttribute("contact").String()
		botAlias := bot.GetBotAttribute("alias").String()
		reply := "Hi, I'm "
		reply += strings.Title(botName) + ", a staff robot. I see you've asked for help.\n\n"
		reply += "I've been programmed to perform a variety of tasks for your team, and will respond when you send me commands that match specific patterns. " +
			"For instance, you can activate my built-in help function by sending a message like this:\n\n"
		reply += botName + ", help ping\nor:\nhelp ping, " + botName + "\n\n... which would give you help on my ping command.\n\n"
		if len(botAlias) > 0 {
			reply += "To save a little typing, you can also address messages to me by prefixing it with my alias ( " + botAlias + " ), like this:\n\n" + botAlias + "help ping\n\n"
		}
		reply += "When the help text for a command has (something) in parentheses, that term or phrase is optional. If <something> is in angle brackets, it's required. With the help function, if you don't supply a keyword you will get help for every command available to you in the current channel, which may differ between channels depending on each channel's purpose. If you use a keyword, you will get help for every command with a matching keyword, along with the channels where it can be used. If the help text is too long, I'll send you a direct message so the channels don't fill up with help output.\n\n"
		reply += "Also, from time to I may ask you a question, prompting for additional information - these messages will mention you by name if not in a private conversation. You only need to type your reply - if you address me by name (or alias), I'll consider it a new command and send an error to the plugin requesting input. Additionally, there are two special replies I understand: \"=\" means 'use the default value', whatever that might be; \"-\" means 'cancel', returning an error value to the plugin.\n\n"
		reply += "For basic information about me, you can use my \"info\" command. Finally, if there's anything else you'd like to see me do, please contact my administrator"
		if len(botContact) > 0 {
			reply += ", " + botContact + "."
		} else {
			reply += "."
		}
		if m.Channel != "" {
			bot.Reply("I've sent you a private message introducing myself")
		}
		bot.SendUserMessage(m.User, reply)
	} else if command == "catchall" {
		bot.Reply("Sorry, that didn't match any commands I know, or may refer to a command that's not available in this channel; try '%s, help <keyword>'", botName)
	}
	return
}

func init() {
	bot.RegisterPlugin("help", robot.PluginHandler{Handler: help})
}
