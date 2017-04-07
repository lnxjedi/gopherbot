package bot

// Handle SIGINT and SIGTERM with a graceful shutdown

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		pluginsRunning.Lock()
		pluginsRunning.shuttingDown = true
		pluginsRunning.Unlock()
		Log(Info, fmt.Sprintf("Received signal: %s, shutting down gracefully", sig))
		// Wait for all plugins to stop running
		pluginsRunning.Wait()
		// Stop the brain after it finishes any current task
		brainQuit()
		Log(Info, fmt.Sprintf("Exiting on signal: %s", sig))
		close(finish)
	}()
}
