package bot

// WaitForBackgroundInits blocks until the current plugin init batch has
// finished. Integration harnesses use this to keep startup/reload init work
// from leaking events into message assertions.
func WaitForBackgroundInits() {
	waitForPluginInitQuiescence()
}
