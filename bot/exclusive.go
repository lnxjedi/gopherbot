package bot

import (
	"sync"
)

var runQueues = struct {
	m map[string][]chan struct{}
	sync.Mutex
}{
	make(map[string][]chan struct{}),
	sync.Mutex{},
}

// Exclusive lets a pipeline insure the same pipeline isn't running twice
// in parallel, to prevent jobs from stomping on each other when it's not safe.
// The string argument ("" allowed) is appended to the pipeline namespace
// to allow for greater granularity; e.g. two builds of different packages
// could use the same pipeline and run safely together, but if it's the same
// package the pipeline might want to queue or just abort. The queueTask
// argument indicates whether the pipeline should queue up to be restarted,
// or abort at the end of the current task. When Exclusive returns false,
// the current task should exit.
// If the task requests queueing, it will either re-run (if the lock holder
// has finished) or queue up (if not) when the current task exits. When it's
// ready to run, the task will be started from the beginning with the same
// arguments, holding the Exclusive lock, and the call to Exclusive will
// always succeed.
// The safest way to use Exclusive is near the beginning of a pipeline.
func (r *Robot) Exclusive(tag string, queueTask bool) (success bool) {
	c := r.getContext()
	task, _, _ := getTask(c.currentTask)
	if task.PrivateNameSpace {
		tag = task.NameSpace + ":" + tag
	} else {
		tag = c.nameSpace + ":" + tag
	}
	c.exclusiveTag = tag
	runQueues.Lock()
	_, exists := runQueues.m[tag]
	if !exists {
		// Take the lock
		runQueues.m[tag] = []chan struct{}{}
		c.exclusive = true
		success = true
		runQueues.Unlock()
		return
	}
	runQueues.Unlock()
	// Update state to indicate what to do after callTask()
	if queueTask {
		c.exclusive = true
		c.queueTask = true
	} else {
		c.abortPipeline = true
	}
	return
}
