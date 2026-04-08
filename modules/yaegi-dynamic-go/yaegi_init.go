//go:build !test

package yaegidynamicgo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lnxjedi/gopherbot/robot"
)

// Initialize sets up the Yaegi environment.
func Initialize(handler robot.Handler) (err error) {
	homePath := os.Getenv("GOPHER_HOME")
	if homePath == "" {
		homePath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	installPath := handler.GetInstallPath()
	if installPath == "" {
		return fmt.Errorf("handler returned empty install path")
	}
	configFull := handler.GetConfigPath()
	if configFull == "" {
		return fmt.Errorf("handler returned empty config path")
	}

	robotInstallPath := filepath.Join(installPath, "robot")
	installLibPath := filepath.Join(installPath, "lib")
	configLibPath := filepath.Join(configFull, "lib")
	goPath, err = ensureGoPath(homePath, robotInstallPath, installLibPath, configLibPath)
	if err != nil {
		return err
	}

	// Successful initialization; the deferred function will handle logging
	return nil
}
