package robot

import (
	"log"
	"sync"
)

type BrainProviderRegistration struct {
	Provider func(Handler) SimpleBrain
}

var brainProviderRegistry = struct {
	sync.RWMutex
	registrations map[string]BrainProviderRegistration
}{
	registrations: make(map[string]BrainProviderRegistration),
}

// RegisterSimpleBrain allows brain providers to register themselves with the
// shared engine/provider contract surface.
func RegisterSimpleBrain(name string, provider func(Handler) SimpleBrain) {
	brainProviderRegistry.Lock()
	defer brainProviderRegistry.Unlock()

	validateNameOrFatal(name)

	if _, exists := brainProviderRegistry.registrations[name]; exists {
		log.Fatalf("Brain provider '%s' is already registered", name)
	}
	brainProviderRegistry.registrations[name] = BrainProviderRegistration{
		Provider: provider,
	}
}

func GetBrainProviderRegistration(name string) (BrainProviderRegistration, bool) {
	brainProviderRegistry.RLock()
	defer brainProviderRegistry.RUnlock()
	registration, ok := brainProviderRegistry.registrations[name]
	return registration, ok
}

func ListBrainProviderRegistrations() map[string]BrainProviderRegistration {
	brainProviderRegistry.RLock()
	defer brainProviderRegistry.RUnlock()
	out := make(map[string]BrainProviderRegistration, len(brainProviderRegistry.registrations))
	for name, registration := range brainProviderRegistry.registrations {
		out[name] = registration
	}
	return out
}
