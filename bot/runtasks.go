package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lnxjedi/robot"
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
	if state.shuttingDown && task.name != "builtin-admin" {
		state.RUnlock()
		Log(robot.Warn, "Not starting new pipeline for task '%s', shutting down", task.name)
		return robot.RobotStopping
	}
	state.RUnlock()
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
		c.environment["GOPHER_CONFIGDIR"] = configFull
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

	initChannel := w.Channel
	// A job or plugin is always the first task in a pipeline; a new
	// sub-pipeline is created if a job is added in another pipeline.
	if isJob {
		// Job parameters are available to the whole pipeline, plugin
		// parameters are not.
		for _, p := range task.Parameters {
			_, exists := c.environment[p.Name]
			if !exists {
				c.environment[p.Name] = p.Value
			}
		}
		if len(task.NameSpace) > 0 {
			if ns, ok := w.tasks.nameSpaces[task.NameSpace]; ok {
				for _, p := range ns.Parameters {
					_, exists := c.environment[p.Name]
					if !exists {
						c.environment[p.Name] = p.Value
					}
				}
			} else {
				Log(robot.Error, "NameSpace '%s' not found for task '%s' (this should never happen)", task.NameSpace, task.name)
			}
		}
		// TODO / NOTE: RawMsg will differ between plugins and triggers - document?
		// histories use the job name for maximum separation
		c.jobName = task.name
		// Exclusive always uses the pipeline nameSpace, regardless of the task that calls it
		c.nameSpace = getNameSpace(task)
		c.environment["GOPHER_JOB_NAME"] = c.jobName
		c.environment["GOPHER_START_CHANNEL"] = w.Channel
		// To change the channel to the job channel, we need to clear the ProcotolChannel
		w.Channel = task.Channel
		w.ProtocolChannel = ""
	}
	c.environment["GOPHER_PIPE_NAME"] = task.name
	// Once Active, we need to use the Mutex for access to some fields; see
	// pipeContext/type pipeContext
	w.registerActive(parent)
	rememberRuns := 0
	if isJob {
		rememberRuns = job.HistoryLogs
	}
	w.Lock()
	pipeHistory, link, ref, idx := newLogger(c.pipeName, w.eid, "", w.id, rememberRuns)
	c.histName = c.pipeName
	c.runIndex = idx
	c.environment["GOPHER_RUN_INDEX"] = fmt.Sprintf("%d", idx)
	c.logger = pipeHistory
	var logref string
	if rememberRuns > 0 {
		if len(link) > 0 && len(ref) > 0 {
			c.environment["GOPHER_LOG_LINK"] = link
			c.environment["GOPHER_LOG_REF"] = ref
			logref = fmt.Sprintf(" (log %s; link %s)", ref, link)
		} else if len(link) > 0 {
			// TODO: this case should never happen; verify and remove?
			c.environment["GOPHER_LOG_LINK"] = link
			logref = fmt.Sprintf(" (link %s)", link)
		} else if len(ref) > 0 {
			c.environment["GOPHER_LOG_REF"] = ref
			logref = fmt.Sprintf(" (log %s)", ref)
		}
	}
	w.Unlock()
	if isJob && (!job.Quiet || c.verbose) {
		r := w.makeRobot()
		taskinfo := task.name
		if len(args) > 0 {
			taskinfo += " " + strings.Join(args, " ")
		}
		schannel := initChannel
		if schannel == "" {
			schannel = "(direct message)"
		}
		switch ptype {
		case jobTrigger:
			r.Say("Starting job '%s', run %d%s - triggered by app '%s' in channel '%s'", taskinfo, c.runIndex, logref, w.User, schannel)
		case jobCommand:
			r.Say("Starting job '%s', run %d%s - requested by user '%s' in channel '%s'", taskinfo, c.runIndex, logref, w.User, schannel)
		case spawnedTask:
			r.Say("Starting job '%s', run %d%s - spawned by pipeline '%s': %s", taskinfo, c.runIndex, logref, ppipeName, ppipeDesc)
		case scheduled:
			r.Say("Starting scheduled job '%s', run %d%s", taskinfo, c.runIndex, logref)
		default:
			r.Say("Starting job '%s', run %d%s", taskinfo, c.runIndex, logref)
		}
		c.verbose = true
	}

	ts := TaskSpec{task.name, command, args, t}
	c.nextTasks = []TaskSpec{ts}

	var errString string
	ret, errString = w.runPipeline(primaryTasks, ptype, true)
	c.environment["GOPHER_FINAL_TASK"] = c.taskName
	finalTask := c.taskName
	c.environment["GOPHER_FINAL_TYPE"] = c.taskType
	finalType := c.taskType
	if c.taskType == "plugin" {
		c.environment["GOPHER_FINAL_COMMAND"] = c.plugCommand
	}
	c.environment["GOPHER_FINAL_ARGS"] = strings.Join(c.taskArgs, " ")
	c.environment["GOPHER_FINAL_DESC"] = c.taskDesc
	finalDesc := c.taskDesc
	numFailTasks := len(w.failTasks)
	if ret != robot.Normal {
		// Add a default tail-log for simple jobs
		if isJob && !job.Quiet && numFailTasks == 0 {
			tailtask := w.tasks.getTaskByName("tail-log")
			sendtask := w.tasks.getTaskByName("send-message")
			w.failTasks = []TaskSpec{
				{
					Name:      "send-message",
					Command:   "run",
					Arguments: []string{fmt.Sprintf("pipeline failed in task %s with exit code %d (%s); log excerpt:", c.taskName, ret, ret)},
					task:      sendtask,
				},
				{
					Name:    "tail-log",
					Command: "run",
					task:    tailtask,
				},
			}
			numFailTasks = 2
		}
		c.section("failed", fmt.Sprintf("pipeline failed in task %s with exit code %d (%s)", c.taskName, ret, ret))
		fc := int64(ret)
		c.environment["GOPHER_FAIL_CODE"] = strconv.FormatInt(fc, 10)
		c.environment["GOPHER_FAIL_STR"] = ret.String()
	} else {
		c.section("done", "primary pipeline has completed")
	}
	// Close the log so final / fail tasks could potentially send log emails / links
	c.logger.Close()

	numFinalTasks := len(w.finalTasks)
	if numFinalTasks > 0 {
		w.runPipeline(finalTasks, ptype, false)
	}
	if ret != robot.Normal {
		if numFailTasks > 0 {
			w.runPipeline(failTasks, ptype, false)
		}
	}
	// Release logs that shouldn't be saved
	c.logger.Finalize()

	if isPlugin && ret != robot.Normal {
		if !w.automaticTask && errString != "" {
			w.Reply(errString)
		}
	}
	if isJob && !job.Quiet {
		r := w.makeRobot()
		if ret == robot.Normal {
			r.Say("Finished job '%s', run %d, final task '%s', status: normal", c.pipeName, c.runIndex, finalTask)
		} else {
			var td string
			if len(c.taskDesc) > 0 {
				td = " - " + finalDesc
			}
			jobName := c.pipeName
			if len(c.nsExtension) > 0 {
				jobName += ":" + c.nsExtension
			}
			if ret == robot.PipelineAborted {
				r.Say("Job '%s', run number %d aborted, exclusive job '%s' already in progress", jobName, c.runIndex, c.exclusiveTag)
			} else {
				r.Say("Job '%s', run number %d failed in %s: '%s'%s, exit code: %d (%s)", jobName, c.runIndex, finalType, finalTask, td, int(ret), ret)
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
	w.deregister()
	// Once deregistered, no Robot can get a pointer to the worker, and
	// locking is no longer needed. Invalid calls to getLockedWorker()
	// will log an error and return nil.
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

		w.taskName = task.name
		w.taskDesc = task.Description
		w.plugCommand = ""
		w.taskArgs = args
		if isJob {
			w.taskType = "job"
		} else if isPlugin {
			w.taskType = "plugin"
			w.plugCommand = command
		} else {
			w.taskType = "task"
		}

		// Security checks for jobs & plugins
		if (isJob || isPlugin) && !w.automaticTask {
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
				if command != "init" {
					emit(CommandTaskRan)
				}
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
			case jobCommand:
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
		}
		if w.stage == finalTasks && ret != robot.Normal {
			w.finalFailed = append(w.finalFailed, task.name)
		}
		// All tasks in final and fail pipelines always run
		if w.stage != finalTasks && w.stage != failTasks && ret != robot.Normal {
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
					if (isJob && !job.Quiet) || ptype == jobCommand {
						w.makeRobot().Say("Queueing task '%s' in pipeline '%s'", task.name, w.pipeName)
					}
					// Now we block until kissed by a Handsome Prince
					<-wakeUp
					Log(robot.Debug, "Bot #%d in queue waking up and re-starting task '%s'", w.id, task.name)
					if (job != nil && !job.Quiet) || ptype == jobCommand {
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

// getEnvironment generates the environment for each task run.
func (w *worker) getEnvironment(t interface{}) map[string]string {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	c := w.pipeContext
	envhash := make(map[string]string)
	// Start with the pipeline environment; values configured for the job,
	// or set with SetParameter(name, value). Unprivileged plugins don't
	// get pipe env.
	pipeEnv := true
	if isPlugin && !task.Privileged {
		pipeEnv = false
	}
	if pipeEnv && len(c.environment) > 0 {
		for k, v := range c.environment {
			envhash[k] = v
		}
	}
	// These values are always fixed
	envhash["GOPHER_CHANNEL"] = w.Channel
	envhash["GOPHER_USER"] = w.User
	envhash["GOPHER_PROTOCOL"] = strings.ToLower(fmt.Sprintf("%s", w.Protocol))
	envhash["GOPHER_TASK_NAME"] = c.taskName
	envhash["GOPHER_PIPELINE_TYPE"] = c.ptype.String()
	envhash["GOPHER_CALLER_ID"] = w.eid
	envhash["GOPHER_HTTP_POST"] = "http://" + listenPort
	envhash["GOPHER_INSTALLDIR"] = installPath
	// Configured parameters for a pipeline task don't apply if already set;
	// task parameters are effectively default values if not otherwise
	// provided.
	for _, p := range task.Parameters {
		_, exists := envhash[p.Name]
		if !exists {
			envhash[p.Name] = p.Value
		}
	}
	// Next lowest prio are namespace params; task parameters can override
	// parameters from the namespace.
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
