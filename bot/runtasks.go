package bot

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
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
// jobcommands: checkJobMatchersAndRun or scheduledTask,
// runPipeline.
func (c *botContext) startPipeline(parent *botContext, t interface{}, ptype pipelineType, command string, args ...string) (ret TaskRetVal) {
	task, _, job := getTask(t)
	isJob := job != nil
	ppipeName := c.pipeName
	ppipeDesc := c.pipeDesc
	c.pipeName = task.name
	c.pipeDesc = task.Description
	// TODO: Replace the waitgroup, pluginsRunning, defer func(), etc.
	robot.Add(1)
	robot.Lock()
	robot.pluginsRunning++
	c.timeZone = robot.timeZone
	robot.Unlock()
	defer func() {
		robot.Lock()
		robot.pluginsRunning--
		// TODO: this check shouldn't be necessary; remove and test
		if robot.pluginsRunning >= 0 {
			robot.Done()
		}
		robot.Unlock()
	}()

	if isJob {
		// TODO / NOTE: RawMsg will differ between plugins and triggers - document?
		c.jobName = task.name // Exclusive always uses the jobName, regardless of the task that calls it
		c.history = robot.history
		c.workingDirectory = robot.workSpace
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
			iChannel := r.Channel
			r.Channel = job.Channel
			switch ptype {
			case jobTrigger:
				c.Channel = job.Channel // send bot output of triggered jobs to job channel
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
	ts := taskSpec{task.name, command, args, t}
	c.nextTasks = []taskSpec{ts}

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
			r.Say(fmt.Sprintf("Job '%s', run number %d failed in task: '%s'%s", jobName, c.runIndex, c.failedTaskName, td))
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
	var p []taskSpec
	eventEmitted := false

	switch c.stage {
	case primaryTasks:
		p = c.nextTasks
		c.nextTasks = []taskSpec{}
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
			c.debug(fmt.Sprintf("Running task with command '%s' and arguments: %v", command, args), false)
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
					c.nextTasks = []taskSpec{}
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
				c.nextTasks = []taskSpec{}
				// the case where bot.queueTask is true is handled right after
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

// callTask does the real work of running a job or plugin with a command and arguments.
func (bot *botContext) callTask(t interface{}, command string, args ...string) (errString string, retval TaskRetVal) {
	bot.currentTask = t
	r := bot.makeRobot()
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	// This should only happen in the rare case that a configured authorizer or elevator is disabled
	if task.Disabled {
		msg := fmt.Sprintf("callTask failed on disabled task %s; reason: %s", task.name, task.reason)
		Log(Error, msg)
		bot.debug(msg, false)
		return msg, ConfigurationError
	}
	if bot.logger != nil {
		var desc string
		if len(task.Description) > 0 {
			desc = fmt.Sprintf("Starting task: %s", task.Description)
		} else {
			desc = "Starting task"
		}
		bot.logger.Section(task.name, desc)
	}

	if !(task.name == "builtInadmin" && command == "abort") {
		defer checkPanic(r, fmt.Sprintf("Plugin: %s, command: %s, arguments: %v", task.name, command, args))
	}
	Log(Debug, fmt.Sprintf("Dispatching command '%s' to task '%s' with arguments '%#v'", command, task.name, args))

	// Set up the per-task environment
	envhash := make(map[string]string)
	if len(bot.environment) > 0 {
		for k, v := range bot.environment {
			envhash[k] = v
		}
	}
	// Pull stored and configured env vars specific to this task and supply to
	// this task only. No effect if already defined. Useful mainly for specific
	// tasks to have secrets passed in but not handed to everything in the
	// pipeline.
	storedEnv := make(map[string]string)
	spk := paramPrefix + task.NameSpace
	if len(bot.nsExtension) > 0 {
		spk += ":" + bot.nsExtension
	}

	// Look up stored parameters (mostly secrets) for namespace and extended
	// namespace, and place in environment if not already there.
	_, exists, _ := checkoutDatum(paramPrefix+task.NameSpace, &storedEnv, false)
	if exists {
		for key, value := range storedEnv {
			// Dynamically provided and configured parameters take precedence over stored parameters
			_, exists := envhash[key]
			if !exists {
				envhash[key] = value
			}
		}
	}
	if len(bot.nsExtension) > 0 {
		storedEnv := make(map[string]string)
		key := paramPrefix + task.NameSpace + ":" + bot.nsExtension
		_, exists, _ := checkoutDatum(key, &storedEnv, false)
		Log(Debug, fmt.Sprintf("Checking for stored parameter '%s', found: %t", key, exists))
		if exists {
			for key, value := range storedEnv {
				// Dynamically provided and configured parameters take precedence over stored parameters
				_, exists := envhash[key]
				if !exists {
					envhash[key] = value
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
		bot.taskenvironment = envhash
		ret := pluginHandlers[task.name].Handler(r, command, args...)
		bot.taskenvironment = nil
		return "", ret
	}
	var fullPath string // full path to the executable
	var err error
	fullPath, err = getTaskPath(task)
	if err != nil {
		emit(ExternalTaskBadPath)
		return fmt.Sprintf("Error getting path for %s: %v", task.name, err), MechanismFail
	}
	winInterpreter := false
	interpreter, err := getInterpreter(fullPath)
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
		externalArgs = append(externalArgs, fullPath)
	}
	// jobs and tasks don't take a 'command' (it's just 'run', a dummy value)
	if isPlugin {
		externalArgs = append(externalArgs, command)
	}
	externalArgs = append(externalArgs, args...)
	if winInterpreter {
		externalArgs = fixInterpreterArgs(interpreter, externalArgs)
	}
	Log(Debug, fmt.Sprintf("Calling '%s' with interpreter '%s' and args: %q", fullPath, interpreter, externalArgs))
	var cmd *exec.Cmd
	if winInterpreter {
		cmd = exec.Command(interpreter, externalArgs...)
	} else {
		cmd = exec.Command(fullPath, externalArgs...)
	}
	bot.Lock()
	bot.taskName = task.name
	bot.taskDesc = task.Description
	bot.osCmd = cmd
	bot.Unlock()

	envhash["GOPHER_CHANNEL"] = bot.Channel
	envhash["GOPHER_USER"] = bot.User
	envhash["GOPHER_PROTOCOL"] = fmt.Sprintf("%s", bot.Protocol)
	envhash["GOPHER_WORKSPACE"] = robot.workSpace
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
	cmd.Dir = bot.workingDirectory
	Log(Debug, fmt.Sprintf("Running '%s' with environment vars: '%s'", fullPath, strings.Join(keys, "', '")))
	var stderr, stdout io.ReadCloser
	// hold on to stderr in case we need to log an error
	stderr, err = cmd.StderrPipe()
	if err != nil {
		Log(Error, fmt.Errorf("Creating stderr pipe for external command '%s': %v", fullPath, err))
		errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
		return errString, MechanismFail
	}
	if bot.logger == nil {
		// close stdout on the external plugin...
		cmd.Stdout = nil
	} else {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			Log(Error, fmt.Errorf("Creating stdout pipe for external command '%s': %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			return errString, MechanismFail
		}
	}
	if err = cmd.Start(); err != nil {
		Log(Error, fmt.Errorf("Starting command '%s': %v", fullPath, err))
		errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
		return errString, MechanismFail
	}
	if command != "init" {
		emit(ExternalTaskRan)
	}
	if bot.logger == nil {
		var stdErrBytes []byte
		if stdErrBytes, err = ioutil.ReadAll(stderr); err != nil {
			Log(Error, fmt.Errorf("Reading from stderr for external command '%s': %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			return errString, MechanismFail
		}
		stdErrString := string(stdErrBytes)
		if len(stdErrString) > 0 {
			Log(Warn, fmt.Errorf("Output from stderr of external command '%s': %s", fullPath, stdErrString))
			errString = fmt.Sprintf("There was error output while calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			emit(ExternalTaskStderrOutput)
		}
	} else {
		closed := make(chan struct{})
		hl := bot.logger
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				bot.logger.Log("OUT " + line)
			}
			closed <- struct{}{}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				bot.logger.Log("ERR " + line)
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
		if bot.logger != hl {
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
			Log(Error, fmt.Errorf("Waiting on external command '%s': %v", fullPath, err))
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

func getTaskPath(task *botTask) (string, error) {
	if len(task.Path) == 0 {
		err := fmt.Errorf("Path empty for external task: %s", task.name)
		Log(Error, err)
		return "", err
	}
	var fullPath string
	if byte(task.Path[0]) == byte("/"[0]) {
		fullPath = task.Path
		_, err := os.Stat(fullPath)
		if err == nil {
			Log(Debug, "Using fully specified path to plugin:", fullPath)
			return fullPath, nil
		}
		err = fmt.Errorf("Invalid path for external plugin: %s (%v)", fullPath, err)
		Log(Error, err)
		return "", err
	}
	if len(configPath) > 0 {
		_, err := os.Stat(configPath + "/" + task.Path)
		if err == nil {
			fullPath = configPath + "/" + task.Path
			Log(Debug, "Using external plugin from configPath:", fullPath)
			return fullPath, nil
		}
	}
	_, err := os.Stat(installPath + "/" + task.Path)
	if err == nil {
		fullPath = installPath + "/" + task.Path
		Log(Debug, "Using stock external plugin:", fullPath)
		return fullPath, nil
	}
	err = fmt.Errorf("Couldn't locate external plugin %s: %v", task.name, err)
	Log(Error, err)
	return "", err
}

// emulate Unix script convention by calling external scripts with
// an interpreter.
func getInterpreter(scriptPath string) (string, error) {
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

func getExtDefCfg(task *botTask) (*[]byte, error) {
	var fullPath string
	var err error
	if fullPath, err = getTaskPath(task); err != nil {
		return nil, err
	}
	var cfg []byte
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		var interpreter string
		interpreter, err = getInterpreter(fullPath)
		if err != nil {
			err = fmt.Errorf("looking up interpreter for %s: %s", fullPath, err)
			return nil, err
		}
		args := fixInterpreterArgs(interpreter, []string{fullPath, "configure"})
		Log(Debug, fmt.Sprintf("Calling '%s' with args: %q", interpreter, args))
		cmd = exec.Command(interpreter, args...)
	} else {
		Log(Debug, fmt.Sprintf("Calling '%s' with arg: configure", fullPath))
		//cfg, err = exec.Command(fullPath, "configure").Output()
		cmd = exec.Command(fullPath, "configure")
	}
	cmd.Env = []string{fmt.Sprintf("GOPHER_INSTALLDIR=%s", installPath)}
	cfg, err = cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("Problem retrieving default configuration for external plugin '%s', skipping: '%v', output: %s", fullPath, err, exitErr.Stderr)
		} else {
			err = fmt.Errorf("Problem retrieving default configuration for external plugin '%s', skipping: '%v'", fullPath, err)
		}
		return nil, err
	}
	return &cfg, nil
}
