// File: robot/registrations.go
package robot

import (
	"log"
	"regexp"
	"sync"
)

// Define the structures for plugin, job, and task handlers.
// These should match the existing definitions in your codebase.

// Registrations holds all the registered plugins, jobs, and tasks.
type Registrations struct {
	Plugins map[string]PluginHandler
	Jobs    map[string]JobHandler
	Tasks   map[string]*TaskRegistration
}

// TaskRegistration holds the details for a registered task.
type TaskRegistration struct {
	Privileged bool
	Handler    TaskHandler
}

// Internal variables to store registrations and enforce single retrieval.
var (
	registrations = &Registrations{
		Plugins: make(map[string]PluginHandler),
		Jobs:    make(map[string]JobHandler),
		Tasks:   make(map[string]*TaskRegistration),
	}
	registrationsCalled = false
	registrationsMutex  sync.Mutex
)

// Regular expression to validate names.
var identifierRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// Names that are reserved and cannot be registered.
var reservedNames = map[string]bool{
	"bot": true,
}

// RegisterPlugin allows plugins to register themselves.
func RegisterPlugin(name string, handler PluginHandler) {
	registrationsMutex.Lock()
	defer registrationsMutex.Unlock()

	if registrationsCalled {
		log.Fatalf("Attempted to register plugin '%s' after registrations have been processed", name)
	}

	validateNameOrFatal(name)

	if _, exists := registrations.Plugins[name]; exists {
		log.Fatalf("Plugin '%s' is already registered", name)
	}
	if isNameUsed(name) {
		log.Fatalf("Plugin name '%s' collides with an existing job or task", name)
	}

	registrations.Plugins[name] = handler
}

// RegisterJob allows jobs to register themselves.
func RegisterJob(name string, handler JobHandler) {
	registrationsMutex.Lock()
	defer registrationsMutex.Unlock()

	if registrationsCalled {
		log.Fatalf("Attempted to register job '%s' after registrations have been processed", name)
	}

	validateNameOrFatal(name)

	if _, exists := registrations.Jobs[name]; exists {
		log.Fatalf("Job '%s' is already registered", name)
	}
	if isNameUsed(name) {
		log.Fatalf("Job name '%s' collides with an existing plugin or task", name)
	}

	registrations.Jobs[name] = handler
}

// RegisterTask allows tasks to register themselves.
func RegisterTask(name string, privRequired bool, handler TaskHandler) {
	registrationsMutex.Lock()
	defer registrationsMutex.Unlock()

	if registrationsCalled {
		log.Fatalf("Attempted to register task '%s' after registrations have been processed", name)
	}

	validateNameOrFatal(name)

	if _, exists := registrations.Tasks[name]; exists {
		log.Fatalf("Task '%s' is already registered", name)
	}
	if isNameUsed(name) {
		log.Fatalf("Task name '%s' collides with an existing plugin or job", name)
	}

	registrations.Tasks[name] = &TaskRegistration{
		Privileged: privRequired,
		Handler:    handler,
	}
}

// GetRegistrations returns all collected registrations.
// It can only be called once; subsequent calls will return nil.
func GetRegistrations() *Registrations {
	registrationsMutex.Lock()
	defer registrationsMutex.Unlock()

	if registrationsCalled {
		return nil
	}
	registrationsCalled = true
	return registrations
}

// Helper function to validate a name and exit if invalid.
func validateNameOrFatal(name string) {
	if !identifierRe.MatchString(name) {
		log.Fatalf("Name '%s' doesn't match the required pattern '%s'", name, identifierRe.String())
	}
	if reservedNames[name] {
		log.Fatalf("Name '%s' is reserved and cannot be registered", name)
	}
}

// Helper function to check if a name is already used.
func isNameUsed(name string) bool {
	_, pluginExists := registrations.Plugins[name]
	_, jobExists := registrations.Jobs[name]
	_, taskExists := registrations.Tasks[name]
	return pluginExists || jobExists || taskExists
}

// The following is an example of how a plugin might use the registration functions.
// func init() {
//     // Example plugin registration.
//     RegisterPlugin("examplePlugin", PluginHandler{
//         Handler: func(r Robot, command string, args ...string) TaskRetVal {
//             // Plugin handler code.
//             return Ok
//         },
//         Config: nil, // Optional configuration.
//     })

//     // Example job registration.
//     RegisterJob("exampleJob", JobHandler{
//         Handler: func(r Robot, args ...string) TaskRetVal {
//             // Job handler code.
//             return Ok
//         },
//     })

//     // Example task registration.
//     RegisterTask("exampleTask", false, TaskHandler{
//         Handler: func(r Robot, args ...string) TaskRetVal {
//             // Task handler code.
//             return Ok
//         },
//     })
// }
