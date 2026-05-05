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

// see robot/robot.go
func (r Robot) Exclusive(tag string, queueTask bool) (success bool) {
	if r.pipeContext == nil || r.currentTask == nil {
		Log(robot.Error, "Exclusive called with no current task")
		return false
	}
	w := getLockedWorker(r.tid)
	if w == nil {
		Log(robot.Error, "Exclusive called without an active worker")
		return false
	}
	defer w.Unlock()
	if len(tag) == 0 {
		Log(robot.Error, "Exclusive called with empty tag")
		return false
	}
	_, plugin, job := getTask(r.currentTask)
	isPlugin := plugin != nil
	isJob := job != nil
	// The intent here is that simple tasks can only call exclusive in the
	// context of a job pipeline, which is the only case where r.nameSpace
	// is set.
	ns := r.nameSpace
	if !isPlugin && !isJob && queueTask && len(ns) == 0 {
		w.Log(robot.Error, "Exclusive called by job or task with queueing outside of job pipeline")
		return false
	}
	if len(ns) == 0 {
		w.Unlock()
		ns = w.getNameSpace(r.currentTask)
		w.Lock()
	}
	lockTag := ns + ":" + tag
	if w.exclusive {
		if w.exclusiveTag != lockTag {
			Log(robot.Error, "Exclusive called with tag '%s' while holding lock for tag '%s'", lockTag, w.exclusiveTag)
			return false
		}
		return true
	}
	w.exclusiveTag = lockTag
	runQueues.Lock()
	_, exists := runQueues.m[lockTag]
	if !exists {
		// Take the lock
		Log(robot.Debug, "Exclusive lock '%s' immediately acquired in pipeline '%s', bot #%d", lockTag, w.pipeName, w.id)
		runQueues.m[lockTag] = []chan struct{}{}
		w.exclusive = true
		success = true
		runQueues.Unlock()
		return
	}
	runQueues.Unlock()
	// Update state to indicate what to do after callTask()
	if queueTask {
		Log(robot.Debug, "Worker #%d requesting queueing", w.id)
		w.queueTask = true
	} else {
		// When a plugin requests Exclusive, it should handle false
		if plugin != nil {
			Log(robot.Debug, "Worker #%d requesting abort, exclusive lock failed", w.id)
			w.abortPipeline = true
		}
	}
	return
}
