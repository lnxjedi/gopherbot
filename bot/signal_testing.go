// +build test

package bot

import (
	"os"
	"os/signal"
	"runtime"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

func sigHandle(sigBreak chan struct{}) {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, unix.SIGINT, unix.SIGTERM)

loop:
	for {
		select {
		case sig := <-sigs:
			buf := make([]byte, 65536)
			ss := runtime.Stack(buf, true)
			os.Stdout.Write(buf[0:ss])
			os.Stdout.Write([]byte("\n"))
			panic("Tests terminated by signal: " + sig.String())

		// done declared globally at top of this file
		case <-sigBreak:
			Log(robot.Info, "Stopping signal handler")
			break loop
		}
	}
}

// sigHandler for pid 1
func initSigHandle(c *os.Process) {
}
