package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

var envPassThrough = []string{
	"HOME",
	"HOSTNAME",
	"LANG",
	"PATH",
	"USER",
}

// startPipeline is triggered by plugins, job triggers, scheduled tasks, and child jobs
// Called from dispatch: checkPluginMatchersAndRun,
// jobcommands: checkJobMatchersAndRun or ScheduledTask,
// runPipeline.
func (w *worker) startPipeline(parent *worker, t interface{}, ptype pipelineType, command string, args ...string) (ret robot.TaskRetVal) {
	task, plugin, job := getTask(t)
	state.RLock()
	if state.shuttingDown {
		state.RUnlock()
		Log(robot.Warn, "Not starting new pipeline for task '%s', shutting down", task.name)
		return robot.RobotStopping
	}
	state.RUnlock()
	raiseThreadPriv(fmt.Sprintf("new pipeline for task %s / %s", task.name, command))
	isJob := job != nil
	isPlugin := plugin != nil
	var ppipeName, ppipeDesc string
	if parent != nil {
		parent.Lock()
		ppipeName = parent.pipeName
		ppipeDesc = parent.pipeDesc
		parent.Unlock()
	}
	// NOTE: we don't need to worry about locking until the pipeline actually starts
	c := &pipeContext{
		environment: make(map[string]string),
	}
	w.pipeContext = c
	c.pipeName = task.name
	c.pipeDesc = task.Description
	if isPlugin {
		c.privileged = plugin.Privileged
	} else {
		c.privileged = job.Privileged
	}
	if c.privileged {
		if len(homePath) > 0 {
			c.environment["GOPHER_HOME"] = homePath
		}
	}
	// Initial baseDirectory and workingDirectory are the same; SetWorkingDirectory
	// modifies workingDirectory.
	if task.Homed {
		c.baseDirectory = "."
		c.workingDirectory = "."
		c.environment["GOPHER_WORKSPACE"] = w.cfg.workSpace
		c.environment["GOPHER_CONFIGDIR"] = configPath
	} else {
		c.baseDirectory = w.cfg.workSpace
		c.workingDirectory = w.cfg.workSpace
	}
	// Spawned pipelines keep the original ptype
	if c.ptype == unset {
		c.ptype = ptype
	}
	c.timeZone = w.cfg.timeZone
	// redundant but explicit
	c.stage = primaryTasks
	// TODO: Replace the waitgroup, pipelinesRunning, defer func(), etc.
	state.Add(1)
	state.Lock()
	state.pipelinesRunning++
	state.Unlock()
	defer func() {
		state.Lock()
		state.pipelinesRunning--
		// TODO: this check shouldn't be necessary; remove and test
		if state.pipelinesRunning >= 0 {
			state.Done()
		}
		state.Unlock()
	}()

	// A job is always the first task in a pipeline; a new sub-pipeline is created
	// if a job is added in another pipeline.
	if isJob {
		// TODO / NOTE: RawMsg will differ between plugins and triggers - document?
		c.jobName = task.name // Exclusive always uses the jobName, regardless of the task that calls it
		c.environment["GOPHER_JOB_NAME"] = c.jobName
		iChannel := w.Channel
		// To change the channel to the job channel, we need to clear the ProcotolChannel
		w.Channel = task.Channel
		w.ProtocolChannel = ""
		c.history = interfaces.history
		var jh jobHistory
		rememberRuns := job.HistoryLogs
		if rememberRuns == 0 {
			rememberRuns = 1
		}
		key := histPrefix + c.jobName
		tok, _, ret := checkoutDatum(key, &jh, true)
		if ret != robot.Ok {
			Log(robot.Error, "Checking out '%s', no history will be remembered for '%s'", key, c.pipeName)
		} else {
			var start time.Time
			if c.timeZone != nil {
				start = time.Now().In(c.timeZone)
			} else {
				start = time.Now()
			}
			c.runIndex = jh.NextIndex
			c.environment["GOPHER_RUN_INDEX"] = fmt.Sprintf("%d", c.runIndex)
			hist := historyLog{
				LogIndex:   c.runIndex,
				CreateTime: start.Format("Mon Jan 2 15:04:05 MST 2006"),
			}
			jh.NextIndex++
			jh.Histories = append(jh.Histories, hist)
			l := len(jh.Histories)
			if l > rememberRuns {
				jh.Histories = jh.Histories[l-rememberRuns:]
			}
			ret := updateDatum(key, tok, jh)
			if ret != robot.Ok {
				Log(robot.Error, "Updating '%s', no history will be remembered for '%s'", key, c.pipeName)
			} else {
				if job.HistoryLogs > 0 && c.history != nil {
					pipeHistory, err := c.history.NewHistory(c.jobName, hist.LogIndex, job.HistoryLogs)
					if err != nil {
						Log(robot.Error, "Starting history for '%s', no history will be recorded: %v", c.pipeName, err)
					} else {
						c.logger = pipeHistory
					}
				} else {
					if c.history == nil {
						Log(robot.Warn, "Starting history, no history provider available")
					}
				}
			}
		}
		for _, p := range task.Parameters {
			_, exists := c.environment[p.Name]
			if !exists {
				c.environment[p.Name] = p.Value
			}
		}
		if !job.Quiet || c.verbose {
			r := w.makeRobot()
			taskinfo := task.name
			if len(args) > 0 {
				taskinfo += " " + strings.Join(args, " ")
			}
			var link string
			if c.history != nil {
				if url, ok := c.history.GetHistoryURL(task.name, c.runIndex); ok {
					link = fmt.Sprintf(" (link: %s)", url)
				}
			}
			switch ptype {
			case jobTrigger:
				r.Say("Starting job '%s', run %d%s - triggered by app '%s' in channel '%s'", taskinfo, c.runIndex, link, w.User, iChannel)
			case jobCmd:
				r.Say("Starting job '%s', run %d%s - requested by user '%s' in channel '%s'", taskinfo, c.runIndex, link, w.User, iChannel)
			case spawnedTask:
				r.Say("Starting job '%s', run %d%s - spawned by pipeline '%s': %s", taskinfo, c.runIndex, link, ppipeName, ppipeDesc)
			case scheduled:
				r.Say("Starting scheduled job '%s', run %d%s", taskinfo, c.runIndex, link)
			default:
				r.Say("Starting job '%s', run %d%s", taskinfo, c.runIndex, link)
			}
			c.verbose = true
		}
	}

	ts := TaskSpec{task.name, command, args, t}
	c.nextTasks = []TaskSpec{ts}

	// Once Active, we need to use the Mutex for access to some fields; see
	// pipeContext/type pipeContext
	w.registerActive(parent)

	var errString string
	ret, errString = w.runPipeline(primaryTasks, ptype, true)
	// Close the log so final / fail tasks could potentially send log emails / links
	if c.logger != nil {
		c.logger.Section("done", "primary pipeline has completed")
		c.logger.Close()
	}
	numFailTasks := len(w.failTasks)
	numFinalTasks := len(w.finalTasks)
	// Run final and fail (cleanup) tasks
	if ret != robot.Normal {
		if numFailTasks > 0 {
			w.runPipeline(failTasks, ptype, false)
		}
	}
	if numFinalTasks > 0 {
		w.runPipeline(finalTasks, ptype, false)
	}
	w.deregister()
	// Once deregistered, no Robot can get a pointer to the worker, and
	// locking is no longer needed. Invalid calls to getLockedWorker()
	// will log an error and return nil.

	if ret != robot.Normal {
		if !w.automaticTask && errString != "" {
			w.Reply(errString)
		}
	}
	if isJob && (!job.Quiet || ret != robot.Normal) {
		r := w.makeRobot()
		if ret == robot.Normal {
			r.Say("Finished job '%s', run %d, final task '%s', status: %s", c.pipeName, c.runIndex, c.taskName, ret)
		} else {
			var td string
			if len(c.failedTaskDescription) > 0 {
				td = " - " + c.failedTaskDescription
			}
			jobName := c.pipeName
			if len(c.nsExtension) > 0 {
				jobName += ":" + c.nsExtension
			}
			if ret == robot.PipelineAborted {
				r.Say("Job '%s', run number %d aborted, job '%s' already in progress", jobName, c.runIndex, c.exclusiveTag)
			} else {
				r.Say("Job '%s', run number %d failed in task: '%s'%s, exit code: %s", jobName, c.runIndex, c.failedTask, td, ret)
			}
		}
	}
	if c.exclusive {
		tag := c.exclusiveTag
		runQueues.Lock()
		queue, _ := runQueues.m[tag]
		queueLen := len(queue)
		if queueLen == 0 {
			Log(robot.Debug, "Bot #%d finished exclusive pipeline '%s', no waiters in queue, removing", w.id, c.exclusiveTag)
			delete(runQueues.m, tag)
		} else {
			Log(robot.Debug, "Bot #%d finished exclusive pipeline '%s', %d waiters in queue, waking next task", w.id, c.exclusiveTag, queueLen)
			wakeUpTask := queue[0]
			queue = queue[1:]
			runQueues.m[tag] = queue
			// Kiss the Princess
			wakeUpTask <- struct{}{}
		}
		runQueues.Unlock()
	}
	return
}

type pipeStage int

const (
	primaryTasks pipeStage = iota
	finalTasks
	failTasks
)

func (w *worker) runPipeline(stage pipeStage, ptype pipelineType, initialRun bool) (ret robot.TaskRetVal, errString string) {
	var p []TaskSpec
	eventEmitted := false
	w.stage = stage
	switch stage {
	case primaryTasks:
		p = w.nextTasks
		w.nextTasks = []TaskSpec{}
	case finalTasks:
		p = w.finalTasks
	case failTasks:
		p = w.failTasks
	}

	l := len(p)
	for i := 0; i < l; i++ {
		ts := p[i]
		command := ts.Command
		args := ts.Arguments
		t := ts.task
		task, plugin, job := getTask(t)
		isJob := job != nil
		isPlugin := plugin != nil

		// Security checks for jobs & plugins
		if (isJob || isPlugin) && !w.automaticTask && w.stage != finalTasks {
			r := w.makeRobot()
			r.currentTask = t
			task, plugin, _ := getTask(t)
			adminRequired := task.RequireAdmin
			if !adminRequired && (plugin != nil && len(plugin.AdminCommands) > 0) {
				for _, i := range plugin.AdminCommands {
					if command == i {
						adminRequired = true
						break
					}
				}
			}
			w.registerWorker(r.tid)
			if adminRequired {
				if !r.CheckAdmin() {
					r.Say("Sorry, '%s/%s' is only available to bot administrators", task.name, command)
					ret = robot.Fail
					deregisterWorker(r.tid)
					break
				}
			}
			if r.checkAuthorization(w, t, command, args...) != robot.Success {
				ret = robot.Fail
				deregisterWorker(r.tid)
				break
			}
			if !w.elevated {
				eret, _ := r.checkElevation(t, command)
				if eret != robot.Success {
					ret = robot.Fail
					deregisterWorker(r.tid)
					break
				}
			}
			deregisterWorker(r.tid)
		}

		if initialRun && !eventEmitted {
			eventEmitted = true
			switch ptype {
			case plugCommand:
				emit(CommandTaskRan) // for testing, otherwise noop
			case plugMessage:
				emit(AmbientTaskRan)
			case catchAll:
				emit(CatchAllTaskRan)
			case jobTrigger:
				emit(TriggeredTaskRan)
			case spawnedTask:
				emit(SpawnedTaskRan)
			case scheduled:
				emit(ScheduledTaskRan)
			case jobCmd:
				emit(JobTaskRan)
			}
		}
		if isJob && i != 0 {
			child := w.clone()
			ret = child.startPipeline(w, t, ptype, command, args...)
		} else {
			debugT(t, fmt.Sprintf("Running task with command '%s' and arguments: %v", command, args), false)
			errString, ret = w.callTask(t, command, args...)
			debugT(t, fmt.Sprintf("Task finished with return value: %s", ret), false)
			if w.stage != finalTasks && ret != robot.Normal {
				w.failedTask = task.name
				if len(args) > 0 {
					w.failedTask += " " + strings.Join(args, " ")
				}
				w.failedTaskDescription = task.Description
			}
		}
		if w.stage != finalTasks && ret != robot.Normal {
			// task / job in pipeline failed
			break
		}
		if !w.exclusive {
			if w.abortPipeline {
				ret = robot.PipelineAborted
				errString = "Pipeline aborted, exclusive lock failed"
				break
			}
			if w.queueTask {
				w.queueTask = false
				w.exclusive = true
				tag := w.exclusiveTag
				runQueues.Lock()
				queue, exists := runQueues.m[tag]
				if exists {
					wakeUp := make(chan struct{})
					queue = append(queue, wakeUp)
					runQueues.m[tag] = queue
					runQueues.Unlock()
					Log(robot.Debug, "Exclusive task in progress, queueing bot #%d and waiting; queue length: %d", w.id, len(queue))
					if (isJob && !job.Quiet) || ptype == jobCmd {
						w.makeRobot().Say("Queueing task '%s' in pipeline '%s'", task.name, w.pipeName)
					}
					// Now we block until kissed by a Handsome Prince
					<-wakeUp
					Log(robot.Debug, "Bot #%d in queue waking up and re-starting task '%s'", w.id, task.name)
					if (job != nil && !job.Quiet) || ptype == jobCmd {
						w.makeRobot().Say("Re-starting queued task '%s' in pipeline '%s'", task.name, w.pipeName)
					}
					// Decrement the index so this task runs again
					i--
					// Clear tasks added in the last run (if any)
					w.nextTasks = []TaskSpec{}
				} else {
					Log(robot.Debug, "Exclusive lock acquired in pipeline '%s', bot #%d", w.pipeName, w.id)
					runQueues.m[tag] = []chan struct{}{}
					runQueues.Unlock()
				}
			}
		}
		if w.stage == primaryTasks {
			t := len(w.nextTasks)
			if t > 0 {
				if i == l-1 {
					p = append(p, w.nextTasks...)
					l += t
				} else {
					ret, errString = w.runPipeline(stage, ptype, false)
				}
				w.nextTasks = []TaskSpec{}
				// the case where c.queueTask is true is handled right after
				// callTask
				if w.abortPipeline {
					ret = robot.PipelineAborted
					errString = "Pipeline aborted, exclusive lock failed"
					break
				}
				if ret != robot.Normal {
					break
				}
			}
		}
	}
	return
}

func (w *worker) getEnvironment(task *Task) map[string]string {
	c := w.pipeContext
	envhash := make(map[string]string)
	if len(c.environment) > 0 {
		for k, v := range c.environment {
			envhash[k] = v
		}
	}

	envhash["GOPHER_CHANNEL"] = w.Channel
	envhash["GOPHER_USER"] = w.User
	envhash["GOPHER_PROTOCOL"] = fmt.Sprintf("%s", w.Protocol)
	envhash["GOPHER_TASK_NAME"] = c.taskName
	envhash["GOPHER_PIPELINE_TYPE"] = c.ptype.String()
	// Configured parameters for a pipeline task don't apply if already set
	for _, p := range task.Parameters {
		_, exists := envhash[p.Name]
		if !exists {
			envhash[p.Name] = p.Value
		}
	}
	// Next lowest prio are namespace params
	if len(task.NameSpace) > 0 {
		if ns, ok := w.tasks.nameSpaces[task.NameSpace]; ok {
			for _, p := range ns.Parameters {
				_, exists := envhash[p.Name]
				if !exists {
					envhash[p.Name] = p.Value
				}
			}
		} else {
			Log(robot.Error, "NameSpace '%s' not found for task '%s' (this should never happen)", task.NameSpace, task.name)
		}
	}
	// Passed-through environment vars have the lowest priority
	for _, p := range envPassThrough {
		_, exists := envhash[p]
		if !exists {
			if value, ok := os.LookupEnv(p); ok {
				envhash[p] = value
			}
		}
	}
	return envhash
}

// getTaskPath searches configPath and installPath and returns the full path
// to the task.
func getTaskPath(task *Task, workDir string) (tpath string, err error) {
	if len(task.Path) == 0 {
		err := fmt.Errorf("Path empty for external task: %s", task.name)
		Log(robot.Error, err.Error())
		return "", err
	}
	tpath, err = getObjectPath(task.Path)
	if err != nil {
		err = fmt.Errorf("Couldn't locate external plugin %s: %v", task.name, err)
		Log(robot.Error, err.Error())
	}
	if filepath.IsAbs(tpath) {
		return
	}
	tpath, err = filepath.Rel(workDir, tpath)
	return
}
