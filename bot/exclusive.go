package bot

import (
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
)

var runQueues = struct {
	m map[string][]chan struct{}
	sync.Mutex
}{
	make(map[string][]chan struct{}),
	sync.Mutex{},
}

// Exclusive lets a pipeline request exclusive execution, to prevent jobs
// from stomping on each other when it's not safe.
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
	if c.exclusive {
		// TODO: make sure tag matches, or error! Note that it's legit and normal
		// to call Exclusive twice with the same tag name.
		return true
	}
	if len(c.jobName) == 0 {
		r.Log(robot.Error, "Exclusive called on pipeline with no job started")
		return false
	}
	if len(tag) > 0 {
		tag = ":" + tag
	}
	tag = c.jobName + tag
	c.exclusiveTag = tag
	runQueues.Lock()
	_, exists := runQueues.m[tag]
	if !exists {
		// Take the lock
		Log(robot.Debug, "Exclusive lock immediately acquired in pipeline '%s', bot #%d", c.pipeName, c.id)
		runQueues.m[tag] = []chan struct{}{}
		c.exclusive = true
		success = true
		runQueues.Unlock()
		return
	}
	runQueues.Unlock()
	// Update state to indicate what to do after callTask()
	if queueTask {
		Log(robot.Debug, "Bot #%d requesting queueing", c.id)
		c.queueTask = true
	} else {
		Log(robot.Debug, "Bot #%d requesting abort, exclusive lock failed", c.id)
		c.abortPipeline = true
	}
	return
}
