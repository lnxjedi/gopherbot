package bot

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
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
func (c *botContext) startPipeline(parent *botContext, t interface{}, ptype pipelineType, command string, args ...string) (ret TaskRetVal) {
	task, _, job := getTask(t)
	isJob := job != nil
	ppipeName := c.pipeName
	ppipeDesc := c.pipeDesc
	c.pipeName = task.name
	c.pipeDesc = task.Description
	// TODO: Replace the waitgroup, pluginsRunning, defer func(), etc.
	botCfg.Add(1)
	botCfg.Lock()
	botCfg.pluginsRunning++
	c.timeZone = botCfg.timeZone
	workSpace := botCfg.workSpace
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

	if isJob {
		// TODO / NOTE: RawMsg will differ between plugins and triggers - document?
		c.jobName = task.name // Exclusive always uses the jobName, regardless of the task that calls it
		c.jobChannel = task.Channel
		c.history = botCfg.history
		c.workingDirectory = workSpace
		var jh jobHistory
		rememberRuns := job.HistoryLogs
		if rememberRuns == 0 {
			rememberRuns = 1
		}
		key := histPrefix + c.jobName
		tok, _, ret := checkoutDatum(key, &jh, true)
		if ret != Ok {
			Log(Error, fmt.Sprintf("Error checking out '%s', no history will be remembered for '%s'", key, c.pipeName))
		} else {
			var start time.Time
			if c.timeZone != nil {
				start = time.Now().In(c.timeZone)
			} else {
				start = time.Now()
			}
			c.runIndex = jh.NextIndex
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
			if ret != Ok {
				Log(Error, fmt.Sprintf("Error updating '%s', no history will be remembered for '%s'", key, c.pipeName))
			} else {
				if job.HistoryLogs > 0 && c.history != nil {
					pipeHistory, err := c.history.NewHistory(c.jobName, hist.LogIndex, job.HistoryLogs)
					if err != nil {
						Log(Error, fmt.Sprintf("Error starting history for '%s', no history will be recorded: %v", c.pipeName, err))
					} else {
						c.logger = pipeHistory
					}
				} else {
					if c.history == nil {
						Log(Warn, "Error starting history, no history provider available")
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
		if !job.Quiet {
			r := c.makeRobot()
			iChannel := c.Channel    // channel where job was triggered / run
			r.Channel = c.jobChannel // channel where job updates are posted
			switch ptype {
			case jobTrigger:
				r.Say(fmt.Sprintf("Starting job '%s', run %d - triggered by app '%s' in channel '%s'", task.name, c.runIndex, c.User, iChannel))
			case jobCmd:
				r.Say(fmt.Sprintf("Starting job '%s', run %d - requested by user '%s' in channel '%s'", task.name, c.runIndex, c.User, iChannel))
			case spawnedTask:
				r.Say(fmt.Sprintf("Starting job '%s', run %d - spawned by pipeline '%s': %s", task.name, c.runIndex, ppipeName, ppipeDesc))
			case scheduled:
				r.Say(fmt.Sprintf("Starting scheduled job '%s', run %d", task.name, c.runIndex))
			default:
				r.Say(fmt.Sprintf("Starting job '%s', run %d", task.name, c.runIndex))
			}
			c.verbose = true
		}
	}

	// redundant but explicit
	c.stage = primaryTasks
	// Once Active, we need to use the Mutex for access to some fields; see
	// botcontext/type botContext
	c.registerActive(nil)
	ts := TaskSpec{task.name, command, args, t}
	c.nextTasks = []TaskSpec{ts}

	var errString string
	ret, errString = c.runPipeline(ptype, true)
	// Run final and fail (cleanup) tasks
	if ret != Normal {
		if len(c.failTasks) > 0 {
			c.stage = failTasks
			c.runPipeline(ptype, false)
		}
	}
	if len(c.finalTasks) > 0 {
		c.stage = finalTasks
		c.runPipeline(ptype, false)
	}
	if c.logger != nil {
		c.logger.Section("done", "pipeline has completed")
		c.logger.Close()
	}
	c.deregister()
	if ret != Normal {
		if !c.automaticTask && errString != "" {
			c.makeRobot().Reply(errString)
		}
	}
	if isJob && (!job.Quiet || ret != Normal) {
		r := c.makeRobot()
		r.Channel = job.Channel
		if ret == Normal {
			r.Say(fmt.Sprintf("Finished job '%s', run %d, final task '%s', status: %s", c.pipeName, c.runIndex, c.taskName, ret))
		} else {
			var td string
			if len(c.failedTaskDescription) > 0 {
				td = " - " + c.failedTaskDescription
			}
			jobName := c.pipeName
			if len(c.nsExtension) > 0 {
				jobName += ":" + c.nsExtension
			}
			if ret == PipelineAborted {
				r.Say(fmt.Sprintf("Job '%s', run number %d aborted, job '%s' already in progress", jobName, c.runIndex, c.exclusiveTag))
			} else {
				r.Say(fmt.Sprintf("Job '%s', run number %d failed in task: '%s'%s, exit code: %s", jobName, c.runIndex, c.failedTaskName, td, ret))
			}
		}
	}
	if c.exclusive {
		tag := c.exclusiveTag
		runQueues.Lock()
		queue, _ := runQueues.m[tag]
		queueLen := len(queue)
		if queueLen == 0 {
			Log(Debug, fmt.Sprintf("Bot #%d finished exclusive pipeline '%s', no waiters in queue, removing", c.id, c.exclusiveTag))
			delete(runQueues.m, tag)
		} else {
			Log(Debug, fmt.Sprintf("Bot #%d finished exclusive pipeline '%s', %d waiters in queue, waking next task", c.id, c.exclusiveTag, queueLen))
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

func (c *botContext) runPipeline(ptype pipelineType, initialRun bool) (ret TaskRetVal, errString string) {
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
			_, plugin, _ := getTask(t)
			if plugin != nil && len(plugin.AdminCommands) > 0 {
				adminRequired := false
				for _, i := range plugin.AdminCommands {
					if command == i {
						adminRequired = true
						break
					}
				}
				if adminRequired {
					if !r.CheckAdmin() {
						r.Say("Sorry, that command is only available to bot administrators")
						ret = Fail
						break
					}
				}
			}
			if c.checkAuthorization(t, command, args...) != Success {
				ret = Fail
				break
			}
			if !c.elevated {
				eret, required := c.checkElevation(t, command)
				if eret != Success {
					ret = Fail
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
			if c.stage != finalTasks && ret != Normal {
				c.failedTaskName = task.name
				c.failedTaskDescription = task.Description
			}
		}
		if c.stage != finalTasks && ret != Normal {
			// task / job in pipeline failed
			break
		}
		if !c.exclusive {
			if c.abortPipeline {
				ret = PipelineAborted
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
					Log(Debug, fmt.Sprintf("Exclusive task in progress, queueing bot #%d and waiting; queue length: %d", c.id, len(queue)))
					if (isJob && !job.Quiet) || ptype == jobCmd {
						c.makeRobot().Say(fmt.Sprintf("Queueing task '%s' in pipeline '%s'", task.name, c.pipeName))
					}
					// Now we block until kissed by a Handsome Prince
					<-wakeUp
					Log(Debug, fmt.Sprintf("Bot #%d in queue waking up and re-starting task '%s'", c.id, task.name))
					if (job != nil && !job.Quiet) || ptype == jobCmd {
						c.makeRobot().Say(fmt.Sprintf("Re-starting queued task '%s' in pipeline '%s'", task.name, c.pipeName))
					}
					// Decrement the index so this task runs again
					i--
					// Clear tasks added in the last run (if any)
					c.nextTasks = []TaskSpec{}
				} else {
					Log(Debug, fmt.Sprintf("Exclusive lock acquired in pipeline '%s', bot #%d", c.pipeName, c.id))
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
					ret = PipelineAborted
					errString = "Pipeline aborted, exclusive lock failed"
					break
				}
				if ret != Normal {
					break
				}
			}
		}
	}
	return
}

// callTask does the real work of running a job, task or plugin with a command and arguments.
func (c *botContext) callTask(t interface{}, command string, args ...string) (errString string, retval TaskRetVal) {
	c.currentTask = t
	r := c.makeRobot()
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	// This should only happen in the rare case that a configured authorizer or elevator is disabled
	if task.Disabled {
		msg := fmt.Sprintf("callTask failed on disabled task %s; reason: %s", task.name, task.reason)
		Log(Error, msg)
		c.debug(msg, false)
		return msg, ConfigurationError
	}
	if c.logger != nil {
		var taskinfo string
		if isPlugin {
			taskinfo = task.name + " " + command
		} else {
			taskinfo = task.name
		}
		if len(args) > 0 {
			taskinfo += " " + strings.Join(args, " ")
		}
		var desc string
		if len(task.Description) > 0 {
			desc = fmt.Sprintf("Starting task: %s", task.Description)
		} else {
			desc = "Starting task"
		}
		c.logger.Section(taskinfo, desc)
	}

	if !(task.name == "builtInadmin" && command == "abort") {
		defer checkPanic(r, fmt.Sprintf("Plugin: %s, command: %s, arguments: %v", task.name, command, args))
	}
	Log(Debug, fmt.Sprintf("Dispatching command '%s' to task '%s' with arguments '%#v'", command, task.name, args))

	// Set up the per-task environment
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
							Log(Error, fmt.Sprintf("Error decrypting '%s' for task namespace '%s': %v", name, task.NameSpace, err))
							break
						}
						envhash[name] = string(value)
					}
				}
			}
		}
	}

	// Configured parameters for a pipeline task don't apply if already set
	for _, p := range task.Parameters {
		_, exists := envhash[p.Name]
		if !exists {
			envhash[p.Name] = p.Value
		}
	}

	if isPlugin && plugin.taskType == taskGo {
		if command != "init" {
			emit(GoPluginRan)
		}
		Log(Debug, fmt.Sprintf("Call go plugin: '%s' with args: %q", task.name, args))
		c.taskenvironment = envhash
		ret := pluginHandlers[task.name].Handler(r, command, args...)
		c.taskenvironment = nil
		return "", ret
	}
	var taskPath string // full path to the executable
	var err error
	var relpath bool
	taskPath, relpath, err = getTaskPath(task)
	if err != nil {
		emit(ExternalTaskBadPath)
		return fmt.Sprintf("Error getting path for %s: %v", task.name, err), MechanismFail
	}
	winInterpreter := false
	interpreter, err := getInterpreter(taskPath, relpath)
	if err != nil {
		errString = "There was a problem calling an external plugin"
		emit(ExternalTaskBadInterpreter)
		return errString, MechanismFail
	}
	if len(interpreter) == 0 {
		interpreter = "(none)"
	} else {
		if runtime.GOOS == "windows" {
			winInterpreter = true
		}
	}
	externalArgs := make([]string, 0, 5+len(args))
	// on Windows, we exec the interpreter with the script as first arg
	if winInterpreter {
		externalArgs = append(externalArgs, taskPath)
	}
	// jobs and tasks don't take a 'command' (it's just 'run', a dummy value)
	if isPlugin {
		externalArgs = append(externalArgs, command)
	}
	externalArgs = append(externalArgs, args...)
	if winInterpreter {
		externalArgs = fixInterpreterArgs(interpreter, externalArgs)
	}
	Log(Debug, fmt.Sprintf("Calling '%s' with interpreter '%s' and args: %q", taskPath, interpreter, externalArgs))
	var cmd *exec.Cmd
	if winInterpreter {
		cmd = exec.Command(interpreter, externalArgs...)
	} else {
		cmd = exec.Command(taskPath, externalArgs...)
	}
	c.Lock()
	c.taskName = task.name
	c.taskDesc = task.Description
	c.osCmd = cmd
	c.Unlock()

	envhash["GOPHER_CHANNEL"] = c.Channel
	envhash["GOPHER_USER"] = c.User
	envhash["GOPHER_PROTOCOL"] = fmt.Sprintf("%s", c.Protocol)
	// Passed-through environment vars have the lowest priority
	for _, p := range envPassThrough {
		_, exists := envhash[p]
		if !exists {
			// Note that we even pass through empty vars - any harm?
			envhash[p] = os.Getenv(p)
		}
	}
	env := make([]string, 0, len(envhash))
	keys := make([]string, 0, len(envhash))
	for k, v := range envhash {
		if len(k) == 0 {
			Log(Error, fmt.Sprintf("Empty Name value while populating environment for '%s', skipping", task.name))
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", k, v))
		keys = append(keys, k)
	}
	cmd.Env = env
	if relpath {
		cmd.Dir = configPath
	} else {
		cmd.Dir = c.workingDirectory
	}
	Log(Debug, fmt.Sprintf("Running '%s' with environment vars: '%s'", taskPath, strings.Join(keys, "', '")))
	var stderr, stdout io.ReadCloser
	// hold on to stderr in case we need to log an error
	stderr, err = cmd.StderrPipe()
	if err != nil {
		Log(Error, fmt.Errorf("Creating stderr pipe for external command '%s': %v", taskPath, err))
		errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
		return errString, MechanismFail
	}
	if c.logger == nil {
		// close stdout on the external plugin...
		cmd.Stdout = nil
	} else {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			Log(Error, fmt.Errorf("Creating stdout pipe for external command '%s': %v", taskPath, err))
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			return errString, MechanismFail
		}
	}
	if err = cmd.Start(); err != nil {
		Log(Error, fmt.Errorf("Starting command '%s': %v", taskPath, err))
		errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
		return errString, MechanismFail
	}
	if command != "init" {
		emit(ExternalTaskRan)
	}
	if c.logger == nil {
		var stdErrBytes []byte
		if stdErrBytes, err = ioutil.ReadAll(stderr); err != nil {
			Log(Error, fmt.Errorf("Reading from stderr for external command '%s': %v", taskPath, err))
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			return errString, MechanismFail
		}
		stdErrString := string(stdErrBytes)
		if len(stdErrString) > 0 {
			Log(Warn, fmt.Errorf("Output from stderr of external command '%s': %s", taskPath, stdErrString))
			errString = fmt.Sprintf("There was error output while calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			emit(ExternalTaskStderrOutput)
		}
	} else {
		closed := make(chan struct{})
		hl := c.logger
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				c.logger.Log("OUT " + line)
			}
			closed <- struct{}{}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				c.logger.Log("ERR " + line)
			}
			closed <- struct{}{}
		}()
		halfClosed := false
	closeLoop:
		for {
			select {
			case <-closed:
				if halfClosed {
					break closeLoop
				}
				halfClosed = true
			}
		}
		if c.logger != hl {
			hl.Close()
		}
	}
	if err = cmd.Wait(); err != nil {
		retval = Fail
		success := false
		if exitstatus, ok := err.(*exec.ExitError); ok {
			if status, ok := exitstatus.Sys().(syscall.WaitStatus); ok {
				retval = TaskRetVal(status.ExitStatus())
				if retval == Success {
					success = true
				}
			}
		}
		if !success {
			Log(Error, fmt.Errorf("Waiting on external command '%s': %v", taskPath, err))
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			emit(ExternalTaskErrExit)
		}
	}
	return errString, retval
}

// Windows argument parsing is all over the map; try to fix it here
// Currently powershell only
func fixInterpreterArgs(interpreter string, args []string) []string {
	ire := regexp.MustCompile(`.*[\/\\!](.*)`)
	var i string
	imatch := ire.FindStringSubmatch(interpreter)
	if len(imatch) == 0 {
		i = interpreter
	} else {
		i = imatch[1]
	}
	switch i {
	case "powershell", "powershell.exe":
		for i := range args {
			args[i] = strings.Replace(args[i], " ", "` ", -1)
			args[i] = strings.Replace(args[i], ",", "`,", -1)
			args[i] = strings.Replace(args[i], ";", "`;", -1)
			if args[i] == "" {
				args[i] = "''"
			}
		}
	}
	return args
}

// getTaskPath searches configPath and installPath and returns a path
// to the task. If the path is relative, the bool is true
func getTaskPath(task *BotTask) (tpath string, relpath bool, err error) {
	if len(task.Path) == 0 {
		err := fmt.Errorf("Path empty for external task: %s", task.name)
		Log(Error, err)
		return "", false, err
	}
	var taskPath string
	if path.IsAbs(task.Path) {
		taskPath = task.Path
		_, err := os.Stat(taskPath)
		if err == nil {
			Log(Debug, "Using fully specified path to plugin:", taskPath)
			return taskPath, false, nil
		}
		err = fmt.Errorf("Invalid path for external plugin: %s (%v)", taskPath, err)
		Log(Error, err)
		return "", false, err
	}
	if len(configPath) > 0 {
		taskPath = path.Join(configPath, task.Path)
		_, err := os.Stat(taskPath)
		if err == nil {
			// The one case where relpath is true
			Log(Debug, "Using external plugin from configPath:", taskPath)
			return task.Path, true, nil
		}
	}
	if _, err := os.Stat(installPath + "/" + task.Path); err == nil {
		taskPath = installPath + "/" + task.Path
		Log(Debug, "Using stock external plugin:", taskPath)
		return taskPath, false, nil
	}
	err = fmt.Errorf("Couldn't locate external plugin %s: %v", task.name, err)
	Log(Error, err)
	return "", false, err
}

// emulate Unix script convention by calling external scripts with
// an interpreter.
func getInterpreter(spath string, relpath bool) (string, error) {
	var scriptPath string
	if relpath {
		scriptPath = path.Join(configPath, spath)
	} else {
		scriptPath = spath
	}
	if _, err := os.Stat(scriptPath); err != nil {
		err = fmt.Errorf("file stat: %s", err)
		Log(Error, fmt.Sprintf("Error getting interpreter for %s: %s", scriptPath, err))
		return "", err
	}
	script, err := os.Open(scriptPath)
	if err != nil {
		err = fmt.Errorf("opening file: %s", err)
		Log(Warn, fmt.Sprintf("Unable to get interpreter for %s: %s", scriptPath, err))
		return "", nil
	}
	r := bufio.NewReader(script)
	iline, err := r.ReadString('\n')
	if err != nil {
		err = fmt.Errorf("reading first line: %s", err)
		Log(Debug, fmt.Sprintf("Problem getting interpreter for %s - %s", scriptPath, err))
		return "", nil
	}
	if !strings.HasPrefix(iline, "#!") {
		err := fmt.Errorf("Interpreter not found for %s; first line doesn't start with '#!'", scriptPath)
		Log(Debug, err)
		return "", nil
	}
	iline = strings.TrimRight(iline, "\n\r")
	interpreter := strings.TrimPrefix(iline, "#!")
	Log(Debug, fmt.Sprintf("Detected interpreter for %s: %s", scriptPath, interpreter))
	return interpreter, nil
}

func getExtDefCfg(task *BotTask) (*[]byte, error) {
	var taskPath string
	var err error
	var relpath bool
	if taskPath, relpath, err = getTaskPath(task); err != nil {
		return nil, err
	}
	var cfg []byte
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		var interpreter string
		interpreter, err = getInterpreter(taskPath, relpath)
		if err != nil {
			err = fmt.Errorf("looking up interpreter for %s: %s", taskPath, err)
			return nil, err
		}
		args := fixInterpreterArgs(interpreter, []string{taskPath, "configure"})
		Log(Debug, fmt.Sprintf("Calling '%s' with args: %q", interpreter, args))
		cmd = exec.Command(interpreter, args...)
	} else {
		Log(Debug, fmt.Sprintf("Calling '%s' with arg: configure", taskPath))
		//cfg, err = exec.Command(taskPath, "configure").Output()
		cmd = exec.Command(taskPath, "configure")
	}
	if relpath {
		cmd.Dir = configPath
	}
	cmd.Env = []string{fmt.Sprintf("GOPHER_INSTALLDIR=%s", installPath)}
	cfg, err = cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("Problem retrieving default configuration for external plugin '%s', skipping: '%v', output: %s", taskPath, err, exitErr.Stderr)
		} else {
			err = fmt.Errorf("Problem retrieving default configuration for external plugin '%s', skipping: '%v'", taskPath, err)
		}
		return nil, err
	}
	return &cfg, nil
}
