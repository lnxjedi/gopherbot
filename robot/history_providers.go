package robot

import (
	"log"
	"sync"
)

type HistoryProviderRegistration struct {
	Provider func(Handler) HistoryProvider
}

var historyProviderRegistry = struct {
	sync.RWMutex
	registrations map[string]HistoryProviderRegistration
}{
	registrations: make(map[string]HistoryProviderRegistration),
}

// RegisterHistoryProvider allows history providers to register themselves with
// the shared engine/provider contract surface.
func RegisterHistoryProvider(name string, provider func(Handler) HistoryProvider) {
	historyProviderRegistry.Lock()
	defer historyProviderRegistry.Unlock()

	validateNameOrFatal(name)

	if _, exists := historyProviderRegistry.registrations[name]; exists {
		log.Fatalf("History provider '%s' is already registered", name)
	}
	historyProviderRegistry.registrations[name] = HistoryProviderRegistration{
		Provider: provider,
	}
}

func GetHistoryProviderRegistration(name string) (HistoryProviderRegistration, bool) {
	historyProviderRegistry.RLock()
	defer historyProviderRegistry.RUnlock()
	registration, ok := historyProviderRegistry.registrations[name]
	return registration, ok
}

func ListHistoryProviderRegistrations() map[string]HistoryProviderRegistration {
	historyProviderRegistry.RLock()
	defer historyProviderRegistry.RUnlock()
	out := make(map[string]HistoryProviderRegistration, len(historyProviderRegistry.registrations))
	for name, registration := range historyProviderRegistry.registrations {
		out[name] = registration
	}
	return out
}
