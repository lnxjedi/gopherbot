package bot

// Elevator plugins provide an elevate method for checking if the user
// can run a privileged command.

import "log"

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
}
