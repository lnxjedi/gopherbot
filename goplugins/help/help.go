// help spits out a helpful message when a user just types "help" in
// a channel. Advanced users will probably write their own.
package help

import (
	"github.com/parsley42/gopherbot/bot"
)

var (
	gobot   bot.Robot
	botName string
)

// Define the handler function
func help(bot bot.Robot, command string, args ...string) {
	if command == "help" { // user just typed 'help' - the robot should introduce itself
		botName := bot.GetAttribute("name")
		botFullName := bot.GetAttribute("fullName")
		botContact := bot.GetAttribute("contact")
		botAlias := bot.GetAttribute("alias")
		reply := "Hi, I'm "
		if len(botFullName) > 0 {
			reply += botFullName + " - but you should just call me " + botName + ".\n"
		} else {
			reply += botName + ".\n"
		}
		reply += "I'm one of the staff robots available to your team, and can respond in different ways to messages that match specific patterns. " +
			"For instance, I will give you help on some of the things I can do when you send me a message that matches the word \"help\" followed by an optional keyword. " +
			"You can address messages to me by using a direct message, or in a channel like this: \"" + botName + ", help (keyword)\". For instance:\n`"
		reply += botName + ", help ping`\nor:\n`help ping, " + botName + "`\nwould give you help on my ping command.\n"
		if len(botAlias) > 0 {
			reply += "To save a little typing, you can also direct a message to me by prefixing it with my alias (" + botAlias + "), like this:\n`" + botAlias + "help ping`\n"
		}
		reply += "When you see (something) in parentheses, that term or phrase is optional. If <something> is in angle brackets, it's required. With the help function, if you don't supply a keyword you will get help for every command available to you in the current channel. If you use a keyword, you will get help for every command with a matching keyword, along with the channels where it can be used. If the help text is too long, I'll send you a direct message so the channels don't fill up with help output.\n"
		reply += "Additionally, some messages (like a bare 'help') will trigger commands as well, and help may or may not be available for those.\n\nFinally, if there's anything else you'd like to see me do, please contact my administrator"
		if len(botContact) > 0 {
			reply += ", " + botContact + "."
		} else {
			reply += "."
		}
		bot.SendUserMessage(reply)
	}
}

func init() {
	bot.RegisterPlugin("help", help)
}
