package bot

import (
	"testing"
	"time"
)

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

	taskLookup.Lock()
	taskLookup.i[tid] = w
	taskLookup.Unlock()
	t.Cleanup(func() {
		taskLookup.Lock()
		delete(taskLookup.i, tid)
		taskLookup.Unlock()
	})

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
	r := Robot{
		pipeContext: &pipeContext{
			currentTask:  task,
			nameSpace:    "job-ns",
			exclusive:    true,
			exclusiveTag: "job-ns:qa-CLONE",
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
