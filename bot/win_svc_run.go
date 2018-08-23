// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package bot

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

type myservice struct{}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	// Start the robot
	stopped := run()
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
				Log(Info, "Shutting down on Windows service stop / shutdown")
				botCfg.Lock()
				if botCfg.shuttingDown {
					botCfg.Unlock()
					eventLog.Warning(1, "Received Windows service stop / shutdown while shutdown in progress")
				} else {
					botCfg.shuttingDown = true
					if botCfg.pluginsRunning > 0 {
						runningCount := botCfg.pluginsRunning
						botCfg.Unlock()
						eventLog.Warning(1, fmt.Sprintf("Stop/shutdown requested with %d plugins running; waiting for all plugins to finish", runningCount))
					} else {
						botCfg.Unlock()
					}
					stop()
				}
			case svc.Pause:
				botCfg.Lock()
				botCfg.paused = true
				botCfg.Unlock()
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				botCfg.Lock()
				botCfg.paused = false
				botCfg.Unlock()
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				eventLog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		case <-stopped:
			break loop
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string) {
	var err error
	eventLog, err = eventlog.Open(name)
	if err != nil {
		log.Println("Failed to open eventlog")
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
