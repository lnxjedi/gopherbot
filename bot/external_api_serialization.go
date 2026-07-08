package bot

import (
	"errors"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

func workerForTaskID(tid int) *worker {
	if tid == 0 {
		return nil
	}
	taskLookup.RLock()
	w := taskLookup.i[tid]
	taskLookup.RUnlock()
	return w
}

func workerForRobotAPI(r robot.Robot) *worker {
	switch rr := r.(type) {
	case Robot:
		return workerForTaskID(rr.tid)
	case *Robot:
		if rr == nil {
			return nil
		}
		return workerForTaskID(rr.tid)
	default:
		return nil
	}
}

type externalAPICallGate interface {
	beginSerializedExternalAPICall() bool
	finishSerializedExternalAPICall()
}

func externalAPICallGateForRobotAPI(r robot.Robot) externalAPICallGate {
	if gate, ok := r.(externalAPICallGate); ok {
		return gate
	}
	return workerForRobotAPI(r)
}

func killExternalProcessGroup(pid int) timeoutInterruptResult {
	if pid == 0 {
		return timeoutInterruptResult{manual: true}
	}
	if err := unix.Kill(-pid, unix.SIGKILL); err != nil && !errors.Is(err, unix.ESRCH) {
		return timeoutInterruptResult{pid: pid, err: err}
	}
	return timeoutInterruptResult{killed: true, pid: pid}
}

func (w *worker) externalProcessPIDLocked() int {
	if w == nil || w.pipeContext == nil || w.osCmd == nil || w.osCmd.Process == nil {
		return 0
	}
	return w.osCmd.Process.Pid
}

func (w *worker) beginSerializedExternalAPICall() bool {
	if w == nil {
		return false
	}
	w.serializeAPICalls.Lock()
	w.Lock()
	pending := w.externalKillPending
	w.Unlock()
	if pending {
		w.killPendingExternalProcessForAPIUnlock()
		w.serializeAPICalls.Unlock()
		return false
	}
	return true
}

func (w *worker) finishSerializedExternalAPICall() {
	if w == nil {
		return
	}
	w.killPendingExternalProcessForAPIUnlock()
	w.serializeAPICalls.Unlock()
}

func (w *worker) requestSerializedExternalKill() timeoutInterruptResult {
	if w == nil {
		return timeoutInterruptResult{manual: true}
	}
	w.Lock()
	w.externalKillPending = true
	w.externalKillResultReady = false
	w.Unlock()

	w.serializeAPICalls.Lock()
	defer w.serializeAPICalls.Unlock()

	w.Lock()
	if w.externalKillResultReady {
		result := w.externalKillResult
		w.externalKillResultReady = false
		w.Unlock()
		return result
	}
	if !w.externalKillPending {
		w.Unlock()
		return timeoutInterruptResult{stale: true}
	}
	w.externalKillPending = false
	pid := w.externalProcessPIDLocked()
	w.Unlock()
	return killExternalProcessGroup(pid)
}

func (w *worker) killPendingExternalProcessForAPIUnlock() {
	w.Lock()
	if !w.externalKillPending {
		w.Unlock()
		return
	}
	w.externalKillPending = false
	pid := w.externalProcessPIDLocked()
	w.Unlock()

	result := killExternalProcessGroup(pid)
	w.Lock()
	w.externalKillResult = result
	w.externalKillResultReady = true
	w.Unlock()
}
