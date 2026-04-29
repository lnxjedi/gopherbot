//go:build test
// +build test

package bot

// WaitForBackgroundInitsForTesting blocks until the current plugin init batch
// has finished. The integration harness uses this to keep startup/reload init
// work from leaking events into message assertions.
func WaitForBackgroundInitsForTesting() {
	WaitForBackgroundInits()
}
