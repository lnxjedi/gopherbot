//go:build test

package yaegidynamicgo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lnxjedi/gopherbot/robot"
)

// Initialize sets up the Yaegi environment for testing.
// It logs informational messages using the provided handler and returns any encountered errors.
func Initialize(handler robot.Handler) (err error) {
	// Deferred function to log successful initialization if no error occurred
	defer func() {
		if err == nil {
			handler.Log(robot.Info, "Yaegi GOPATH set to: %s", goPath)
		}
	}()

	// 1. Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// 2. Set GOPATH to ".gopath" within the current directory
	goPath = filepath.Join(currentDir, ".gopath")

	// 3. If ".gopath" exists, remove it to ensure a clean setup
	if _, err := os.Stat(goPath); err == nil {
		err = os.RemoveAll(goPath)
		if err != nil {
			return fmt.Errorf("failed to remove existing .gopath: %w", err)
		}
	} else if !os.IsNotExist(err) {
		// An error other than "not exists" occurred
		return fmt.Errorf("failed to stat .gopath: %w", err)
	}

	// 4. Define robotSrcPath within the InitializeTest function
	robotSrcPath := filepath.Join(goPath, "src", "github.com", "lnxjedi", "gopherbot", "robot")

	// 5. Create the robot source directory with appropriate permissions
	err = os.MkdirAll(robotSrcPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create robot source directory: %w", err)
	}

	// 6. Define the path where the robot package is installed for testing
	robotInstallPath := filepath.Join(currentDir, "robot")

	// 7. Copy the robot package to GOPATH using the existing copyDir function
	err = copyDir(robotInstallPath, robotSrcPath)
	if err != nil {
		return fmt.Errorf("failed to copy robot package: %w", err)
	}

	// Successful initialization; the deferred function will handle logging
	return nil
}
