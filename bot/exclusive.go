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
	task, plugin, job := getTask(r.currentTask)
	if r.exclusive {
		lockTag := exclusiveLockTag(r.exclusiveNameSpace(task, plugin), tag)
		if r.exclusiveTag != lockTag {
			Log(robot.Error, "Exclusive called with tag '%s' while holding exclusive lock '%s'", lockTag, r.exclusiveTag)
			return false
		}
		return true
	}
	isPlugin := plugin != nil
	isJob := job != nil
	w := getLockedWorker(r.tid)
	if w == nil {
		Log(robot.Error, "Exclusive called without an active worker")
		return false
	}
	ns := r.nameSpace
	// The intent here is that simple tasks can only call exclusive in the
	// context of a job pipeline, which is the only case where r.nameSpace
	// is set.
	if !isPlugin && !isJob && queueTask && len(r.nameSpace) == 0 {
		w.Log(robot.Error, "Exclusive called by job or task with queueing outside of job pipeline")
		w.Unlock()
		return false
	}
	if len(ns) == 0 {
		w.Unlock()
		ns = w.getNameSpace(r.currentTask)
		w.Lock()
	}
	defer w.Unlock()
	tag = exclusiveLockTag(ns, tag)
	w.exclusiveTag = tag
	runQueues.Lock()
	_, exists := runQueues.m[tag]
	if !exists {
		// Take the lock
		Log(robot.Debug, "Exclusive lock '%s' immediately acquired in pipeline '%s', bot #%d", tag, w.pipeName, w.id)
		runQueues.m[tag] = []chan struct{}{}
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

func exclusiveLockTag(ns, tag string) string {
	if len(tag) > 0 {
		tag = ":" + tag
	}
	return ns + tag
}

func (r Robot) exclusiveNameSpace(task *Task, plugin *Plugin) string {
	if len(task.NameSpace) > 0 {
		return task.NameSpace
	}
	if plugin != nil {
		return task.name
	}
	return r.nameSpace
}
