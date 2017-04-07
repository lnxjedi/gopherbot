// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package bot

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

type myservice struct{}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	// Start the connector's main loop in a goroutine
	go conn.Run(finish)
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				pluginsRunning.Lock()
				pluginsRunning.paused = true
				pluginsRunning.Unlock()
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				pluginsRunning.Lock()
				pluginsRunning.paused = false
				pluginsRunning.Unlock()
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				eventLog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	pluginsRunning.Lock()
	pluginsRunning.shuttingDown = true
	if pluginsRunning.count > 0 {
		runningCount := pluginsRunning.count
		pluginsRunning.Unlock()
		eventLog.Warning(1, fmt.Sprintf("Stop/shutdown requested with %d plugins running; waiting for all plugins to finish", runningCount))
	} else {
		pluginsRunning.Unlock()
	}
	// Wait for all plugins to stop running
	pluginsRunning.Wait()
	// Stop the brain after it finishes any current task
	brainQuit()
	Log(Info, "Exiting on administrator command")
	time.Sleep(time.Second)
	close(finish)
	return
}

func runService(name string) {
	var err error
	eventLog, err = eventlog.Open(name)
	if err != nil {
		botLogger.Println("Failed to open eventlog")
		return
	}
	defer eventLog.Close()

	eventLog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	err = run(name, &myservice{})
	if err != nil {
		eventLog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	eventLog.Info(1, fmt.Sprintf("%s service stopped", name))
}
