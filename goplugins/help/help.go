// The help plugin spits out a helpful message when a user just types "help" in
// a channel, and also responds when the user addresses the robot but no plugin
// matched. Advanced users will probably disable this one and write their own.
package help

import (
	"fmt"
	"strings"

	"github.com/uva-its/gopherbot/bot"
)

var (
	gobot   bot.Robot
	botName string
)

// Default plugin configuration for help
const defaultConfig = `
Help:
- Keywords: [ "*" ]
  Helptext: [ "(bot), help help - give general help on the help system and using the robot" ]
CommandMatches:
- Command: help
  Regex: '^(?i:help help)'
MessageMatches:
- Command: help
  Regex: '^(?i:help)$'
AllChannels: true
CatchAll: true
`

// Define the handler function
func help(bot *bot.Robot, command string, args ...string) {
	botName := bot.GetBotAttribute("name").String()
	if command == "help" { // user just typed 'help' - the robot should introduce itself
		botContact := bot.GetBotAttribute("contact").String()
		botAlias := bot.GetBotAttribute("alias").String()
		reply := "Hi, I'm "
		reply += strings.Title(botName) + ", a staff robot. I see you've asked for help.\n\n"
		reply += "I've been programmed to perform a variety of tasks for your team, and will respond to messages that match specific patterns. " +
			"For instance, you can activate my built-in help function by sending me a message that matches \"help (keyword)\".\n\n" +
			"You can address messages to me by sending a direct message, or by speaking to me in a channel, like this: \"" + botName + ", help (keyword)\". For instance:\n`"
		reply += botName + ", help ping`\nor:\n`help ping, " + botName + "`\nwould give you help on my ping command.\n"
		if len(botAlias) > 0 {
			reply += "To save a little typing, you can also address messages to me by prefixing it with my alias ( " + botAlias + " ), like this:\n`" + botAlias + "help ping`\n\n"
		}
		reply += "When help text has (something) in parentheses, that term or phrase is optional. If <something> is in angle brackets, it's required. With the help function, if you don't supply a keyword you will get help for every command available to you in the current channel, which may differ between channels depending on each channel's purpose. If you use a keyword, you will get help for every command with a matching keyword, along with the channels where it can be used. If the help text is too long, I'll send you a direct message so the channels don't fill up with help output.\n"
		reply += "Additionally, some messages (like a bare 'help') will trigger commands as well, and help may or may not be available for those.\n\nFinally, if there's anything else you'd like to see me do, please contact my administrator"
		if len(botContact) > 0 {
			reply += ", " + botContact + "."
		} else {
			reply += "."
		}
		if bot.Channel != "" {
			bot.Reply("I've sent you a private message introducing myself")
		}
		bot.SendUserMessage(bot.User, reply)
	} else if command == "catchall" {
		bot.Reply(fmt.Sprintf("Sorry, that didn't match any commands I know, or may refer to a command that's not available in this channel; try '%s, help <keyword>'", botName))
	}
}

func init() {
	bot.RegisterPlugin("help", bot.PluginHandler{DefaultConfig: defaultConfig, Handler: help})
}
