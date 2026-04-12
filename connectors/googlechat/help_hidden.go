package googlechat

import "strings"

func formatHiddenCommand(botName, input string) string {
	name := strings.TrimSpace(strings.TrimPrefix(botName, "/"))
	if name == "" {
		return ""
	}
	fields := strings.Fields(name)
	if len(fields) > 0 {
		name = fields[0]
	}
	command := strings.TrimSpace(input)
	name = strings.ToLower(name)
	if command == "" {
		return "/" + name
	}
	return "/" + name + " " + command
}

func (gc *googleChatConnector) FormatHiddenCommand(input string) string {
	return formatHiddenCommand(gc.slashCommand, input)
}
