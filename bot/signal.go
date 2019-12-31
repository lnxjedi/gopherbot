// +build !test

package bot

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/lnxjedi/gopherbot/robot"
)

func sigHandle(sigBreak chan struct{}) {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

loop:
	for {
		select {
		case sig := <-sigs:
			state.Lock()
			if state.shuttingDown {
				Log(robot.Warn, "Received SIGINT/SIGTERM while shutdown in progress")
				state.Unlock()
			} else {
				state.shuttingDown = true
				state.Unlock()
				signal.Stop(sigs)
				Log(robot.Info, "Exiting on signal: %s", sig)
				stop()
			}
		// done declared globally at top of this file
		case <-sigBreak:
			Log(robot.Info, "Stopping signal handler")
			break loop
		}
	}
}
