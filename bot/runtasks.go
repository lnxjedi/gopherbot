package bot

import (
	"fmt"
	"os"
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
func (c *botContext) startPipeline(parent *botContext, t interface{}, ptype pipelineType, command string, args ...string) (ret robot.TaskRetVal) {
	task, _, job := getTask(t)
	privCheck(fmt.Sprintf("task %s / %s", task.name, command))
	isJob := job != nil
	ppipeName := c.pipeName
	ppipeDesc := c.pipeDesc
	c.pipeName = task.name
	c.pipeDesc = task.Description
	c.protected = task.Protected
	// Spawned pipelines keep the original ptype
	if c.ptype == unset {
		c.ptype = ptype
	}
	// TODO: Replace the waitgroup, pluginsRunning, defer func(), etc.
	botCfg.Add(1)
	botCfg.Lock()
	botCfg.pluginsRunning++
	c.timeZone = botCfg.timeZone
	botCfg.Unlock()
	defer func() {
		botCfg.Lock()
		botCfg.pluginsRunning--
		// TODO: this check shouldn't be necessary; remove and test
		if botCfg.pluginsRunning >= 0 {
			botCfg.Done()
		}
		botCfg.Unlock()
	}()

	// redundant but explicit
	c.stage = primaryTasks
	// Once Active, we need to use the Mutex for access to some fields; see
	// botcontext/type botContext
	c.registerActive(nil)

	// A job is always the first task in a pipeline; a new sub-pipeline is created
	// if a job is added in another pipeline.
	if isJob {
		// TODO / NOTE: RawMsg will differ between plugins and triggers - document?
		c.jobName = task.name // Exclusive always uses the jobName, regardless of the task that calls it
		c.environment["GOPHER_JOB_NAME"] = c.jobName
		c.jobChannel = task.Channel
		botCfg.RLock()
		c.history = botCfg.history
		botCfg.RUnlock()
		c.workingDirectory = ""
		var jh jobHistory
		rememberRuns := job.HistoryLogs
		if rememberRuns == 0 {
			rememberRuns = 1
		}
		key := histPrefix + c.jobName
		tok, _, ret := checkoutDatum(key, &jh, true)
		if ret != robot.Ok {
			Log(robot.Error, "Error checking out '%s', no history will be remembered for '%s'", key, c.pipeName)
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
				Log(robot.Error, "Error updating '%s', no history will be remembered for '%s'", key, c.pipeName)
			} else {
				if job.HistoryLogs > 0 && c.history != nil {
					pipeHistory, err := c.history.NewHistory(c.jobName, hist.LogIndex, job.HistoryLogs)
					if err != nil {
						Log(robot.Error, "Error starting history for '%s', no history will be recorded: %v", c.pipeName, err)
					} else {
						c.logger = pipeHistory
					}
				} else {
					if c.history == nil {
						Log(robot.Warn, "Error starting history, no history provider available")
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
			r := c.makeRobot()
			iChannel := c.Channel // channel where job was triggered / run
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
				r.SendChannelMessage(c.jobChannel, fmt.Sprintf("Starting job '%s', run %d%s - triggered by app '%s' in channel '%s'", taskinfo, c.runIndex, link, c.User, iChannel))
			case jobCmd:
				r.SendChannelMessage(c.jobChannel, fmt.Sprintf("Starting job '%s', run %d%s - requested by user '%s' in channel '%s'", taskinfo, c.runIndex, link, c.User, iChannel))
			case spawnedTask:
				r.SendChannelMessage(c.jobChannel, fmt.Sprintf("Starting job '%s', run %d%s - spawned by pipeline '%s': %s", taskinfo, c.runIndex, link, ppipeName, ppipeDesc))
			case scheduled:
				r.SendChannelMessage(c.jobChannel, fmt.Sprintf("Starting scheduled job '%s', run %d%s", taskinfo, c.runIndex, link))
			default:
				r.SendChannelMessage(c.jobChannel, fmt.Sprintf("Starting job '%s', run %d%s", taskinfo, c.runIndex, link))
			}
			c.verbose = true
		}
	}

	ts := TaskSpec{task.name, command, args, t}
	c.nextTasks = []TaskSpec{ts}

	var errString string
	ret, errString = c.runPipeline(ptype, true)
	// Close the log so final / fail tasks could potentially send log emails / links
	if c.logger != nil {
		c.logger.Section("done", "primary pipeline has completed")
		c.logger.Close()
	}
	// Run final and fail (cleanup) tasks
	if ret != robot.Normal {
		if len(c.failTasks) > 0 {
			c.stage = failTasks
			c.runPipeline(ptype, false)
		}
	}
	if len(c.finalTasks) > 0 {
		c.stage = finalTasks
		c.runPipeline(ptype, false)
	}
	if ret != robot.Normal {
		if !c.automaticTask && errString != "" {
			c.makeRobot().Reply(errString)
		}
	}
	if isJob && (!job.Quiet || ret != robot.Normal) {
		r := c.makeRobot()
		if ret == robot.Normal {
			r.SendChannelMessage(c.jobChannel, fmt.Sprintf("Finished job '%s', run %d, final task '%s', status: %s", c.pipeName, c.runIndex, c.taskName, ret))
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
				r.SendChannelMessage(c.jobChannel, fmt.Sprintf("Job '%s', run number %d aborted, job '%s' already in progress", jobName, c.runIndex, c.exclusiveTag))
			} else {
				r.SendChannelMessage(c.jobChannel, fmt.Sprintf("Job '%s', run number %d failed in task: '%s'%s, exit code: %s", jobName, c.runIndex, c.failedTask, td, ret))
			}
		}
	}
	c.deregister()
	if c.exclusive {
		tag := c.exclusiveTag
		runQueues.Lock()
		queue, _ := runQueues.m[tag]
		queueLen := len(queue)
		if queueLen == 0 {
			Log(robot.Debug, "Bot #%d finished exclusive pipeline '%s', no waiters in queue, removing", c.id, c.exclusiveTag)
			delete(runQueues.m, tag)
		} else {
			Log(robot.Debug, "Bot #%d finished exclusive pipeline '%s', %d waiters in queue, waking next task", c.id, c.exclusiveTag, queueLen)
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

func (c *botContext) runPipeline(ptype pipelineType, initialRun bool) (ret robot.TaskRetVal, errString string) {
	var p []TaskSpec
	eventEmitted := false

	switch c.stage {
	case primaryTasks:
		p = c.nextTasks
		c.nextTasks = []TaskSpec{}
	case finalTasks:
		p = c.finalTasks
	case failTasks:
		p = c.failTasks
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
		if (isJob || isPlugin) && !c.automaticTask && c.stage != finalTasks {
			r := c.makeRobot()
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
			if adminRequired {
				if !r.CheckAdmin() {
					r.Say("Sorry, '%s/%s' is only available to bot administrators", task.name, command)
					ret = robot.Fail
					break
				}
			}
			if c.checkAuthorization(t, command, args...) != robot.Success {
				ret = robot.Fail
				break
			}
			if !c.elevated {
				eret, required := c.checkElevation(t, command)
				if eret != robot.Success {
					ret = robot.Fail
					break
				}
				if required {
					c.elevated = true
				}
			}
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
			child := c.clone()
			ret = child.startPipeline(c, t, ptype, command, args...)
		} else {
			c.debugT(t, fmt.Sprintf("Running task with command '%s' and arguments: %v", command, args), false)
			errString, ret = c.callTask(t, command, args...)
			c.debug(fmt.Sprintf("Task finished with return value: %s", ret), false)
			if c.stage != finalTasks && ret != robot.Normal {
				c.failedTask = task.name
				if len(args) > 0 {
					c.failedTask += " " + strings.Join(args, " ")
				}
				c.failedTaskDescription = task.Description
			}
		}
		if c.stage != finalTasks && ret != robot.Normal {
			// task / job in pipeline failed
			break
		}
		if !c.exclusive {
			if c.abortPipeline {
				ret = robot.PipelineAborted
				errString = "Pipeline aborted, exclusive lock failed"
				break
			}
			if c.queueTask {
				c.queueTask = false
				c.exclusive = true
				tag := c.exclusiveTag
				runQueues.Lock()
				queue, exists := runQueues.m[tag]
				if exists {
					wakeUp := make(chan struct{})
					queue = append(queue, wakeUp)
					runQueues.m[tag] = queue
					runQueues.Unlock()
					Log(robot.Debug, "Exclusive task in progress, queueing bot #%d and waiting; queue length: %d", c.id, len(queue))
					if (isJob && !job.Quiet) || ptype == jobCmd {
						c.makeRobot().Say("Queueing task '%s' in pipeline '%s'", task.name, c.pipeName)
					}
					// Now we block until kissed by a Handsome Prince
					<-wakeUp
					Log(robot.Debug, "Bot #%d in queue waking up and re-starting task '%s'", c.id, task.name)
					if (job != nil && !job.Quiet) || ptype == jobCmd {
						c.makeRobot().Say("Re-starting queued task '%s' in pipeline '%s'", task.name, c.pipeName)
					}
					// Decrement the index so this task runs again
					i--
					// Clear tasks added in the last run (if any)
					c.nextTasks = []TaskSpec{}
				} else {
					Log(robot.Debug, "Exclusive lock acquired in pipeline '%s', bot #%d", c.pipeName, c.id)
					runQueues.m[tag] = []chan struct{}{}
					runQueues.Unlock()
				}
			}
		}
		if c.stage == primaryTasks {
			t := len(c.nextTasks)
			if t > 0 {
				if i == l-1 {
					p = append(p, c.nextTasks...)
					l += t
				} else {
					ret, errString = c.runPipeline(ptype, false)
				}
				c.nextTasks = []TaskSpec{}
				// the case where c.queueTask is true is handled right after
				// callTask
				if c.abortPipeline {
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

func (c *botContext) getEnvironment(task *BotTask) map[string]string {
	envhash := make(map[string]string)
	if len(c.environment) > 0 {
		for k, v := range c.environment {
			envhash[k] = v
		}
	}
	// Pull stored and configured env vars specific to this task and supply to
	// this task only. No effect if already defined. Useful mainly for specific
	// tasks to have secrets passed in but not handed to everything in the
	// pipeline. Repository secrets are populated in robot.go/ExtendNamespace
	cryptKey.RLock()
	initialized := cryptKey.initialized
	key := cryptKey.key
	cryptKey.RUnlock()
	if initialized {
		taskEnv, teExists := c.storedEnv.TaskParams[task.NameSpace]
		if teExists {
			if initialized {
				for name, encvalue := range taskEnv {
					_, exists := envhash[name]
					if !exists {
						value, err := decrypt(encvalue, key)
						if err != nil {
							Log(robot.Error, "Error decrypting '%s' for task namespace '%s': %v", name, task.NameSpace, err)
							break
						}
						envhash[name] = string(value)
					}
				}
			}
		}
	}

	envhash["GOPHER_CHANNEL"] = c.Channel
	envhash["GOPHER_USER"] = c.User
	envhash["GOPHER_PROTOCOL"] = fmt.Sprintf("%s", c.Protocol)
	envhash["GOPHER_TASK_NAME"] = c.taskName
	envhash["GOPHER_PIPELINE_TYPE"] = c.ptype.String()
	// Configured parameters for a pipeline task don't apply if already set
	for _, p := range task.Parameters {
		_, exists := envhash[p.Name]
		if !exists {
			envhash[p.Name] = p.Value
		}
	}
	// Passed-through environment vars have the lowest priority
	for _, p := range envPassThrough {
		_, exists := envhash[p]
		if !exists {
			// Note that we even pass through empty vars - any harm?
			envhash[p] = os.Getenv(p)
		}
	}
	return envhash
}

// getTaskPath searches configPath and installPath and returns the full path
// to the task.
func getTaskPath(task *BotTask) (tpath string, err error) {
	if len(task.Path) == 0 {
		err := fmt.Errorf("path empty for external task: %s", task.name)
		Log(robot.Error, err.Error())
		return "", err
	}
	tpath, err = getObjectPath(task.Path)
	if err != nil {
		err = fmt.Errorf("couldn't locate external plugin %s: %v", task.name, err)
		Log(robot.Error, err.Error())
	}
	return
}
