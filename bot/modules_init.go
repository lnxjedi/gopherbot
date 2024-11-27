package bot

import (
	"fmt"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
	sshagent "github.com/lnxjedi/gopherbot/v2/modules/ssh-agent"
	sshgithelper "github.com/lnxjedi/gopherbot/v2/modules/ssh-git-helper"
	yaegi "github.com/lnxjedi/gopherbot/v2/modules/yaegi-dynamic-go"
)

func initializeModules(handler robot.Handler) error {
	var wg sync.WaitGroup // WaitGroup to wait for all goroutines to finish
	var mu sync.Mutex     // Mutex to protect access to firstErr
	var firstErr error    // Variable to store the first encountered error

	// List of modules to initialize
	modules := []struct {
		name       string
		initialize func(robot.Handler) error
	}{
		{"ssh-agent", sshagent.Initialize},
		{"ssh-known-hosts-helper", sshgithelper.Initialize},
		{"yaegi-dynamic-go", yaegi.Initialize},
	}

	wg.Add(len(modules)) // Set the number of goroutines to wait for

	// Iterate over each module and launch a goroutine to initialize it
	for _, m := range modules {
		m := m // Capture range variable
		go func(module struct {
			name       string
			initialize func(robot.Handler) error
		}) {
			defer wg.Done() // Signal that this goroutine is done

			handler.RaisePriv(fmt.Sprintf("initializing %s", module.name))

			// Attempt to initialize the module
			if err := module.initialize(handler); err != nil {
				// Lock the mutex before accessing firstErr
				mu.Lock()
				// If no error has been recorded yet, record this error
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to initialize %s: %w", module.name, err)
				}
				mu.Unlock() // Unlock the mutex
			}
		}(m)
	}

	wg.Wait() // Wait for all goroutines to finish

	return firstErr // Return the first encountered error, or nil if all succeeded
}
