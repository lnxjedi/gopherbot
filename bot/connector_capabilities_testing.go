//go:build test

package bot

import (
	"fmt"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
)

// ApplyConnectorCapabilitiesForTesting overrides connector capabilities for the
// duration of a test and returns an idempotent restore function.
func ApplyConnectorCapabilitiesForTesting(overrides map[string]robot.ConnectorCapabilities) (func(), error) {
	original := make(map[string]robot.ConnectorCapabilities, len(connectorCapabilityOverrides))
	for name, capabilities := range connectorCapabilityOverrides {
		original[name] = capabilities
	}

	var once sync.Once
	restore := func() {
		once.Do(func() {
			connectorCapabilityOverrides = original
		})
	}

	for protocol, capabilities := range overrides {
		p := normalizeProtocolName(protocol)
		if _, ok := connectorRegistrationForProtocol(p); !ok {
			restore()
			return nil, fmt.Errorf("connector '%s' is not registered", protocol)
		}
		connectorCapabilityOverrides[p] = capabilities
	}

	return restore, nil
}
