package bot

import (
	"os"
	"strings"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
)

// Package global gopherEnv map to store GOPHER_ environment variables
var gopherEnv = make(map[string]string)

// package global startupEnv keeps a copy of direct startup GOPHER_* environment
// variables for restart handoff.
var startupEnv = make(map[string]string)
var gopherEnvMutex sync.RWMutex

// Copy start-up GOPHER_* environment variables to startupEnv before the early
// re-exec handoff scrubs them from the direct process environment.
func init() {
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "GOPHER_") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				startupEnv[key] = value
			}
		}
	}
}

// scrubEnvironment is called from start.go to scrub all GOPHER_* env vars so they don't propagate
// to child processes.
func scrubEnvironment() {
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "GOPHER_") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				setGopherEnvValue(key, value)
				err := os.Unsetenv(key)
				if err != nil {
					Log(robot.Error, "Failed to unset environment variable '%s': %v\n", key, err)
				} else {
					Log(robot.Info, "scrubbed %s from the environment", key)
				}
			}
		}
	}
}

func setGopherEnvValue(key, value string) {
	gopherEnvMutex.Lock()
	gopherEnv[key] = value
	gopherEnvMutex.Unlock()
}

func setStartupEnvValue(key, value string) {
	gopherEnvMutex.Lock()
	startupEnv[key] = value
	gopherEnvMutex.Unlock()
}

func copyStartupEnv() map[string]string {
	gopherEnvMutex.RLock()
	defer gopherEnvMutex.RUnlock()
	out := make(map[string]string, len(startupEnv))
	for key, value := range startupEnv {
		out[key] = value
	}
	return out
}

// Helper function to set environment variables
func setEnv(key, value string) error {
	if strings.HasPrefix(key, "GOPHER_") {
		setGopherEnvValue(key, value)
		Log(robot.Debug, "Set internal GOPHER_ environment variable '%s'", key)
		return nil
	}
	return os.Setenv(key, value)
}

// Helper function to get environment variables
func getEnv(key string) string {
	gopherEnvMutex.RLock()
	value, exists := gopherEnv[key]
	gopherEnvMutex.RUnlock()
	if exists {
		return value
	}
	return os.Getenv(key)
}

// Helper function to lookup environment variables
func lookupEnv(key string) (string, bool) {
	gopherEnvMutex.RLock()
	value, exists := gopherEnv[key]
	gopherEnvMutex.RUnlock()
	if exists {
		return value, true
	}
	return os.LookupEnv(key)
}
