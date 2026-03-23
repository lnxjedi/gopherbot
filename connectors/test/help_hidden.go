package test

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

func (tc *TestConnector) FormatHiddenCommandExample(input string) string {
	return formatHiddenExample(input)
}

func (tc *TestConnector) HiddenCommandHint() string {
	return "Use '/(bot) <command>' to address a hidden command."
}
