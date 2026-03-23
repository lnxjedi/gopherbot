package robot

import (
	"log"
	"sync"
)

type ConnectorCapabilities struct {
	HiddenCommands bool
}

type ConnectorRegistration struct {
	Initialize   func(Handler, *log.Logger) Connector
	Capabilities ConnectorCapabilities
}

// HiddenCommandFormatter is an optional connector contract for rendering
// connector-specific hidden-command help when connector capabilities indicate
// that hidden commands are supported.
type HiddenCommandFormatter interface {
	FormatHiddenCommandExample(string) string
	HiddenCommandHint() string
}

var connectorRegistry = struct {
	sync.RWMutex
	registrations map[string]ConnectorRegistration
}{
	registrations: make(map[string]ConnectorRegistration),
}

// RegisterConnector allows connectors to register themselves with the shared
// engine/connector contract surface.
func RegisterConnector(name string, initialize func(Handler, *log.Logger) Connector, capabilities ConnectorCapabilities) {
	connectorRegistry.Lock()
	defer connectorRegistry.Unlock()

	validateNameOrFatal(name)

	if _, exists := connectorRegistry.registrations[name]; exists {
		log.Fatalf("Connector '%s' is already registered", name)
	}
	connectorRegistry.registrations[name] = ConnectorRegistration{
		Initialize:   initialize,
		Capabilities: capabilities,
	}
}

func GetConnectorRegistration(name string) (ConnectorRegistration, bool) {
	connectorRegistry.RLock()
	defer connectorRegistry.RUnlock()
	registration, ok := connectorRegistry.registrations[name]
	return registration, ok
}

func ListConnectorRegistrations() map[string]ConnectorRegistration {
	connectorRegistry.RLock()
	defer connectorRegistry.RUnlock()
	out := make(map[string]ConnectorRegistration, len(connectorRegistry.registrations))
	for name, registration := range connectorRegistry.registrations {
		out[name] = registration
	}
	return out
}
