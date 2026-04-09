// Package help - plugin spits out a helpful message when a user just types "help" in
// a channel, and also responds when the user addresses the robot but no plugin
// matched. Advanced users will probably disable this one and write their own.
package help

import (
	"strings"
	"unicode"

	"github.com/lnxjedi/gopherbot/robot"
)

func inlineCode(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	return "`" + strings.ReplaceAll(trimmed, "`", "\\`") + "`"
}

func nameCommandExample(botName, command string) string {
	botName = strings.TrimSpace(botName)
	command = strings.TrimSpace(command)
	if botName == "" {
		return inlineCode(command)
	}
	return inlineCode(botName + ", " + command)
}

func preferredCommandExample(botName, botAlias, command string) string {
	botAlias = strings.TrimSpace(botAlias)
	command = strings.TrimSpace(command)
	if botAlias != "" {
		return inlineCode(botAlias + command)
	}
	return nameCommandExample(botName, command)
}

func displayBotName(botName string) string {
	trimmed := strings.TrimSpace(botName)
	if trimmed == "" {
		return "this robot"
	}
	runes := []rune(trimmed)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func buildHelpReply(botName, botAlias, botContact, protocol string) string {
	displayName := displayBotName(botName)
	lines := []string{
		"**Help**",
		"Hi, I'm " + displayName + ", a staff robot. I see you've asked for help.",
		"",
		"I've been programmed to perform a variety of tasks for your team, and I'll respond when you send me commands that match specific patterns.",
		"",
		"**Getting command help**",
		"Ask me for command help with my name:",
		"- " + nameCommandExample(botName, "help ping"),
	}
	if strings.TrimSpace(botAlias) != "" {
		lines = append(lines,
			"",
			"Or use my alias "+inlineCode(botAlias)+" for shorter commands:",
			"- "+preferredCommandExample(botName, botAlias, "help ping"),
		)
	}
	lines = append(lines,
		"",
		"**Useful discovery commands**",
		"- "+preferredCommandExample(botName, botAlias, "help")+" - quick help and pointers",
		"- "+preferredCommandExample(botName, botAlias, "commands")+" - browse command groups available in this channel",
		"- "+preferredCommandExample(botName, botAlias, "help <keyword>")+" - show the best matches with usage and examples",
		"- "+preferredCommandExample(botName, botAlias, "help-all")+" - detailed help for commands available here",
		"",
		"If I can't match a command, I'll usually suggest the closest matches and the next help command to try.",
		"",
		"**When I ask a follow-up question**",
		"Just reply with the answer.",
		"- `=` uses the default value",
		"- `-` cancels",
	)
	if protocol == "Terminal" {
		lines = append(lines,
			"",
			"Since you're using the terminal connector, you can get simple help for changing the user, channel, and thread by pressing `<return>` with an empty command.",
		)
	}
	infoExample := preferredCommandExample(botName, botAlias, "info")
	closing := "For basic information about me, try " + infoExample + "."
	if strings.TrimSpace(botContact) != "" {
		closing += " If there's anything else you'd like to see me do, please contact my administrator, " + botContact + "."
	} else {
		closing += " If there's anything else you'd like to see me do, please contact my administrator."
	}
	lines = append(lines, "", closing)
	return strings.Join(lines, "\n")
}

// Define the handler function
func help(bot robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	botName := bot.GetBotAttribute("name").String()
	if command == "help" { // user just typed 'help' - the robot should introduce itself
		botContact := bot.GetBotAttribute("contact").String()
		botAlias := bot.GetBotAttribute("alias").String()
		reply := buildHelpReply(botName, botAlias, botContact, bot.GetMessage().Protocol.String())
		bot.MessageFormat(robot.BasicMarkdown).SayThread(reply)
	}
	return
}

func init() {
	robot.RegisterPlugin("help", robot.PluginHandler{Handler: help})
}
