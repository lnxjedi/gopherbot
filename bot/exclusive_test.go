package bot

import (
	"testing"
	"time"
)

func registerExclusiveTestWorker(t *testing.T, tid int, w *worker) {
	t.Helper()
	taskLookup.Lock()
	taskLookup.i[tid] = w
	taskLookup.Unlock()
	t.Cleanup(func() {
		taskLookup.Lock()
		delete(taskLookup.i, tid)
		taskLookup.Unlock()
	})
}

func resetRunQueuesForTest(t *testing.T) {
	t.Helper()
	runQueues.Lock()
	original := runQueues.m
	runQueues.m = make(map[string][]chan struct{})
	runQueues.Unlock()
	t.Cleanup(func() {
		runQueues.Lock()
		runQueues.m = original
		runQueues.Unlock()
	})
}

func TestExclusiveUnsupportedQueuedSimpleTaskReleasesWorkerLock(t *testing.T) {
	task := &Task{name: "simple-task"}
	w := &worker{
		pipeContext: &pipeContext{
			currentTask: task,
			pipeName:    "simple-task",
			taskName:    "simple-task",
		},
	}
	tid := getTaskID()
	registerExclusiveTestWorker(t, tid, w)

	r := Robot{
		tid: tid,
		pipeContext: &pipeContext{
			currentTask: task,
			pipeName:    "simple-task",
			taskName:    "simple-task",
		},
	}

	if r.Exclusive("qa-CLONE", true) {
		t.Fatal("Exclusive returned true for unsupported queued simple task")
	}

	locked := make(chan struct{})
	go func() {
		w.Lock()
		w.Unlock()
		close(locked)
	}()

	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Fatal("Exclusive returned while still holding the worker lock")
	}
}

func TestExclusiveRejectsMismatchedTagWhenAlreadyExclusive(t *testing.T) {
	task := &Task{name: "exclusive-task"}
	w := &worker{
		pipeContext: &pipeContext{
			currentTask:  task,
			pipeName:     "exclusive-task",
			taskName:     "exclusive-task",
			nameSpace:    "job-ns",
			exclusive:    true,
			exclusiveTag: "job-ns:qa-CLONE",
		},
	}
	tid := getTaskID()
	registerExclusiveTestWorker(t, tid, w)

	r := Robot{
		tid: tid,
		pipeContext: &pipeContext{
			currentTask: task,
			nameSpace:   "job-ns",
		},
	}

	if !r.Exclusive("qa-CLONE", false) {
		t.Fatal("Exclusive rejected matching tag while already exclusive")
	}
	if r.Exclusive("qa-COPY", false) {
		t.Fatal("Exclusive accepted mismatched tag while already exclusive")
	}
}

func TestExclusiveFailsWithoutCurrentTask(t *testing.T) {
	if (Robot{}).Exclusive("qa-CLONE", false) {
		t.Fatal("Exclusive succeeded without current task")
	}
}

func TestExclusiveUsesNamespacedRunQueueKey(t *testing.T) {
	resetRunQueuesForTest(t)

	task := &Task{name: "exclusive-task"}
	w := &worker{
		pipeContext: &pipeContext{
			currentTask: task,
			pipeName:    "exclusive-task",
			taskName:    "exclusive-task",
		},
	}
	tid := getTaskID()
	registerExclusiveTestWorker(t, tid, w)

	r := Robot{
		tid: tid,
		pipeContext: &pipeContext{
			currentTask: task,
			nameSpace:   "job-ns",
		},
	}

	if !r.Exclusive("qa-CLONE", false) {
		t.Fatal("Exclusive failed to acquire available lock")
	}

	runQueues.Lock()
	_, namespaced := runQueues.m["job-ns:qa-CLONE"]
	_, raw := runQueues.m["qa-CLONE"]
	runQueues.Unlock()

	if !namespaced {
		t.Fatal("Exclusive did not store lock under namespaced key")
	}
	if raw {
		t.Fatal("Exclusive stored lock under raw tag instead of namespaced key")
	}
}
