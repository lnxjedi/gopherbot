//go:build test

package yaegidynamicgo

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func init() {
	initOnce.Do(func() {
		currentDir, err := os.Getwd()
		if err != nil {
			initErr = fmt.Errorf("failed to get current directory: %w", err)
			return
		}

		goPath = filepath.Join(currentDir, ".gopath")
		if _, err := os.Stat(goPath); err == nil {
			err = os.RemoveAll(goPath)
			if err != nil {
				initErr = fmt.Errorf("failed to remove existing .gopath: %w", err)
				return
			}
		}

		robotSrcPath := filepath.Join(goPath, "src", "github.com", "lnxjedi", "gopherbot", "robot")
		err = os.MkdirAll(robotSrcPath, 0755)
		if err != nil {
			initErr = fmt.Errorf("failed to create robot source directory: %w", err)
			return
		}

		robotInstallPath := filepath.Join(currentDir, "robot")
		err = copyDir(robotInstallPath, robotSrcPath)
		if err != nil {
			initErr = fmt.Errorf("failed to copy robot package: %w", err)
			return
		}

		log.Printf("Yaegi GOPATH set to: %s", goPath)
	})
}
