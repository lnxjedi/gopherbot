// Common symbols needed when being built as a module
// +build module

package terminal

import (
	"log"

	"github.com/lnxjedi/gopherbot/robot"
)

// GetInitializer is the common exported symbol for loadable connector modules.
func GetInitializer() (string, func(robot.Handler, *log.Logger) robot.Connector) {
	return "terminal", Initialize
}
