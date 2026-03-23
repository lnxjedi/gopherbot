package ssh

import (
	"regexp"
	"strings"
)

var helpAddressPrefixRegex = regexp.MustCompile(`^\s*/?\((?:alias|bot)\)(?:[,:])?\s*`)

func formatHiddenExample(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "/(bot)"
	}
	rest := strings.TrimSpace(helpAddressPrefixRegex.ReplaceAllString(trimmed, ""))
	if rest == "" {
		return "/(bot)"
	}
	return "/(bot) " + rest
}

func (sc *sshConnector) FormatHiddenCommandExample(input string) string {
	return formatHiddenExample(input)
}

func (sc *sshConnector) HiddenCommandHint() string {
	return "Use '/(bot) <command>' to address a hidden command."
}
