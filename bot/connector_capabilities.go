package bot

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

var connectorRegistrationOverrides = map[string]robot.ConnectorRegistration{}

func connectorRegistrationForProtocol(protocol string) (robot.ConnectorRegistration, bool) {
	p := normalizeProtocolName(protocol)
	if p == "" {
		return robot.ConnectorRegistration{}, false
	}
	if registration, ok := connectorRegistrationOverrides[p]; ok {
		return registration, true
	}
	return robot.GetConnectorRegistration(p)
}

var connectorCapabilityOverrides = map[string]robot.ConnectorCapabilities{}

func capabilitiesForProtocol(protocol string) robot.ConnectorCapabilities {
	p := normalizeProtocolName(protocol)
	if p == "" {
		return robot.ConnectorCapabilities{}
	}
	if capabilities, ok := connectorCapabilityOverrides[p]; ok {
		return capabilities
	}
	if capabilities, ok := getRuntimeConnectorCapabilities(p); ok {
		return capabilities
	}
	return robot.ConnectorCapabilities{}
}

func hiddenCommandsSupportedForProtocol(protocol string) bool {
	return capabilitiesForProtocol(protocol).HiddenCommands
}

func formatHiddenCommand(protocol, command string) string {
	if !hiddenCommandsSupportedForProtocol(protocol) {
		return ""
	}
	conn := getConnectorForProtocol(protocol)
	if conn == nil {
		return ""
	}
	formatter, ok := conn.(robot.HiddenCommandFormatter)
	if !ok {
		return ""
	}
	return strings.TrimSpace(formatter.FormatHiddenCommand(command))
}

func hiddenCommandHintForProtocol(protocol string) string {
	if !hiddenCommandsSupportedForProtocol(protocol) {
		return ""
	}
	command := formatHiddenCommand(protocol, "<command>")
	if command == "" {
		return ""
	}
	return fmt.Sprintf("Use `%s` to address a hidden command.", command)
}
