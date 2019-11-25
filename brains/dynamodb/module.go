// Common symbols needed when being built as a module
// +build module

package dynamobrain

import (
	"github.com/lnxjedi/gopherbot/robot"
)

// GetBrainProvider is the common exported symbol for loadable connector modules.
func GetBrainProvider() (string, func(robot.Handler) robot.SimpleBrain) {
	return "dynamo", provider
}
