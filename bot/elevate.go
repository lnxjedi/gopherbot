package bot

// Elevator plugins provide an elevate method for checking if the user
// can run a privileged command.

import (
	"fmt"
	"log"
)

// map of registered elevate methods
var elevators = make(map[string]func(Handler) Elevate)

// RegisterElevator allows elevate methods to register themselves.
func RegisterElevator(name string, provider func(Handler) Elevate) {
	if stopRegistrations {
		return
	}
	if elevators[name] != nil {
		log.Fatal("Attempted registration of duplicate elevator name:", name)
	}
	elevators[name] = provider
	// Give the elevator a name that's illegal for normal plugins, so
	// it can use the brain without possibility of a normal plugin using
	// the same namespace in the brain.
	plugName := fmt.Sprintf("elevator-%s", name)
	plugIDNameMap[plugName] = plugName
}
