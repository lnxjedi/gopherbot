package bot

import (
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

func capabilitiesForProtocol(protocol string) robot.ConnectorCapabilities {
	registration, ok := connectorRegistrationForProtocol(protocol)
	if !ok {
		return robot.ConnectorCapabilities{}
	}
	return registration.Capabilities
}

func hiddenCommandsSupportedForProtocol(protocol string) bool {
	return capabilitiesForProtocol(protocol).HiddenCommands
}

func formatHiddenCommandExample(protocol, example string) string {
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
	return strings.TrimSpace(formatter.FormatHiddenCommandExample(example))
}

func hiddenCommandHintForProtocol(protocol string) string {
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
	return strings.TrimSpace(formatter.HiddenCommandHint())
}
