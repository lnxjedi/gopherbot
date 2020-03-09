// +build !test

package bot

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/lnxjedi/robot"
	"golang.org/x/sys/unix"
)

func sigHandle(sigBreak chan struct{}) {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, unix.SIGINT, unix.SIGTERM, unix.SIGUSR1, unix.SIGUSR2)

loop:
	for {
		select {
		case sig := <-sigs:
			switch sig {
			case unix.SIGINT, unix.SIGTERM:
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
			case unix.SIGUSR1:
				buf := make([]byte, 32768)
				runtime.Stack(buf, true)
				log.Printf("%s", buf)
				time.Sleep(2 * time.Second)
				panic("SIGUSR1 received")
			case unix.SIGUSR2:
				Log(robot.Info, "Restarting logfile")
				logRotate("")
				Log(robot.Info, "Log rotated")
			}
		// done declared globally at top of this file
		case <-sigBreak:
			Log(robot.Info, "Stopping signal handler")
			break loop
		}
	}
}

// sigHandler for pid 1
func initSigHandle(c *os.Process) {
	Log(robot.Info, "Starting pid 1 signal handler")
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, unix.SIGINT, unix.SIGTERM)

	for {
		select {
		case sig := <-sigs:
			signal.Stop(sigs)
			Log(robot.Info, "Caught signal '%s', propagating to child pid %d", sig, c.Pid)
			c.Signal(sig)
		}
	}
}
