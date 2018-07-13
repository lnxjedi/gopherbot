package bot

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

// startPipeline is triggered by user commands, job triggers, and scheduled tasks.
// Called from dispatch: checkPluginMatchersAndRun,
// jobcommands: checkJobMatchersAndRun or scheduledTask.
func (bot *botContext) startPipeline(t interface{}, ptype pipelineType, command string, args ...string) {
	task, _, job := getTask(t)
	isJob := job != nil
	verbose := (isJob && job.Verbose) || ptype == jobCmd
	bot.pipeName = task.name
	bot.pipeDesc = task.Description
	bot.nameSpace = task.NameSpace
	if len(task.WorkingDirectory) > 0 {
		dir := task.WorkingDirectory
		if filepath.IsAbs(dir) {
			bot.workingDirectory = filepath.Clean(dir)
		} else {
			h := handler{}
			bot.workingDirectory = filepath.Join(h.GetConfigPath(), dir)
		}
	}
	// TODO: Replace the waitgroup, pluginsRunning, defer func(), etc.
	robot.Add(1)
	robot.Lock()
	robot.pluginsRunning++
	history := robot.history
	tz := robot.timeZone
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
	var runIndex int
	if task.HistoryLogs > 0 || isJob {
		var th taskHistory
		rememberRuns := task.HistoryLogs
		if rememberRuns == 0 {
			rememberRuns = 1
		}
		key := histPrefix + bot.pipeName
		tok, _, ret := checkoutDatum(key, &th, true)
		if ret != Ok {
			Log(Error, fmt.Sprintf("Error checking out '%s', no history will be remembered for '%s'", key, bot.pipeName))
		} else {
			var start time.Time
			if tz != nil {
				start = time.Now().In(tz)
			} else {
				start = time.Now()
			}
			runIndex = th.NextIndex
			hist := historyLog{
				LogIndex:   runIndex,
				CreateTime: start.Format("Mon Jan 2 15:04:05 MST 2006"),
			}
			th.NextIndex++
			th.Histories = append(th.Histories, hist)
			l := len(th.Histories)
			if l > rememberRuns {
				th.Histories = th.Histories[l-rememberRuns:]
			}
			ret := updateDatum(key, tok, th)
			if ret != Ok {
				Log(Error, fmt.Sprintf("Error updating '%s', no history will be remembered for '%s'", key, bot.pipeName))
			} else {
				if task.HistoryLogs > 0 {
					pipeHistory, err := history.NewHistory(bot.pipeName, hist.LogIndex, task.HistoryLogs)
					if err != nil {
						Log(Error, fmt.Sprintf("Error starting history for '%s', no history will be recorded: %v", bot.pipeName, err))
					} else {
						bot.logger = pipeHistory
					}
				}
			}
		}
	}

	// Set up the environment for the pipeline, in order of precedence high-low.
	// Done in reverse order with existence checking because the context may
	// already have dynamically provided environment vars, which are highest
	// precedence. Environment vars are retrievable as environment variables for
	// scripts, or using GetParameter(...) in Go plugins.
	if isJob {
		for _, p := range job.Parameters {
			bot.environment[p.Name] = p.Value
		}
	}
	storedEnv := make(map[string]string)
	// Global environment for pipeline from first task
	_, exists, _ := checkoutDatum(paramPrefix+task.NameSpace, &storedEnv, false)
	if exists {
		for key, value := range storedEnv {
			// Dynamically provided and configured parameters take precedence over stored parameters
			_, exists := bot.environment[key]
			if !exists {
				bot.environment[key] = value
			}
		}
	}
	bot.pipeStarting = true
	for _, p := range envPassThrough {
		_, exists := bot.environment[p]
		if !exists {
			// Note that we even pass through empty vars - any harm?
			bot.environment[p] = os.Getenv(p)
		}
	}

	// Once Active, we need to use the Mutex for access to some fields; see
	// botcontext/type botContext
	bot.registerActive()
	r := bot.makeRobot()
	var errString string
	var ret TaskRetVal
	if verbose {
		r.Say(fmt.Sprintf("Starting job '%s', run %d", task.name, runIndex))
	}
	ts := taskSpec{task.name, command, args, t}
	bot.nextTasks = []taskSpec{ts}
	ret, errString = bot.runPipeline(nextT, ptype, true)
	if ret != Normal {
		if !bot.automaticTask && errString != "" {
			r.Reply(errString)
		}
	}
	bot.deregister()
	if bot.logger != nil {
		bot.logger.Section("done", "pipeline has completed")
		bot.logger.Close()
	}
	if ret == Normal && verbose {
		r.Say(fmt.Sprintf("Finished job '%s', run %d", bot.pipeName, runIndex))
	}
	if ret != Normal && isJob {
		task, _, _ := getTask(t)
		r.Reply(fmt.Sprintf("Job '%s', run number %d failed in task: '%s'", bot.pipeName, runIndex, task.name))
	}
	// Run final (cleanup) tasks
	if len(bot.finalTasks) > 0 {
		bot.runPipeline(finalT, ptype, false)
	}
}

type pipeSelector int

const (
	nextT pipeSelector = iota
	finalT
)

func (bot *botContext) runPipeline(s pipeSelector, ptype pipelineType, initialRun bool) (ret TaskRetVal, errString string) {
	var p []taskSpec
	eventEmitted := false
	switch s {
	case nextT:
		p = bot.nextTasks
		bot.nextTasks = []taskSpec{}
	case finalT:
		p = bot.finalTasks
	}
	l := len(p)
	for i := 0; i < l; i++ {
		ts := p[i]
		command := ts.Command
		args := ts.Arguments
		t := ts.task
		// bypass security checks if flag set, or running final tasks
		if !bot.automaticTask && s != finalT {
			r := bot.makeRobot()
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
			if bot.checkAuthorization(t, command, args...) != Success {
				ret = Fail
				break
			}
			if !bot.elevated {
				eret, required := bot.checkElevation(t, command)
				if eret != Success {
					ret = Fail
					break
				}
				if required {
					bot.elevated = true
				}
			}
		}
		if initialRun && !eventEmitted {
			switch ptype {
			case plugCommand:
				emit(CommandTaskRan) // for testing, otherwise noop
			case plugMessage:
				emit(AmbientTaskRan)
			case catchAll:
				emit(CatchAllTaskRan)
			case jobTrigger:
				emit(TriggeredTaskRan)
			case scheduled:
				emit(ScheduledTaskRan)
			case jobCmd:
				emit(JobTaskRan)
			}
		}
		bot.debug(fmt.Sprintf("Running task with command '%s' and arguments: %v", command, args), false)
		errString, ret = bot.callTask(t, command, args...)
		bot.debug(fmt.Sprintf("Task finished with return value: %s", ret), false)
		if s != finalT && ret != Normal {
			// task in pipeline failed
			break
		}
		if s == nextT {
			t := len(bot.nextTasks)
			if t > 0 {
				if i == l-1 {
					p = append(p, bot.nextTasks...)
					l += t
				} else {
					ret, errString = bot.runPipeline(nextT, ptype, false)
				}
				bot.nextTasks = []taskSpec{}
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
	task, plugin, job := getTask(t)
	isPlugin := plugin != nil
	isJob := job != nil
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
	if isPlugin && plugin.taskType == taskGo {
		if command != "init" {
			emit(GoPluginRan)
		}
		Log(Debug, fmt.Sprintf("Call go plugin: '%s' with args: %q", task.name, args))
		return "", pluginHandlers[task.name].Handler(r, command, args...)
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
	if !bot.pipeStarting {
		storedEnv := make(map[string]string)
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
		// Configured parameters for a pipeline job don't apply if already set
		if isJob {
			for _, p := range job.Parameters {
				_, exists := envhash[p.Name]
				if !exists {
					envhash[p.Name] = p.Value
				}
			}
		}
	} else {
		bot.pipeStarting = false
	}

	envhash["GOPHER_CHANNEL"] = bot.Channel
	envhash["GOPHER_USER"] = bot.User
	envhash["GOPHER_PROTOCOL"] = fmt.Sprintf("%s", bot.Protocol)
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
		errString = fmt.Sprintf("There were errors calling external plugin '%s', you might want to ask an administrator to check the logs", task.name)
		return errString, MechanismFail
	}
	if bot.logger == nil {
		// close stdout on the external plugin...
		cmd.Stdout = nil
	} else {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			Log(Error, fmt.Errorf("Creating stdout pipe for external command '%s': %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external plugin '%s', you might want to ask an administrator to check the logs", task.name)
			return errString, MechanismFail
		}
	}
	if err = cmd.Start(); err != nil {
		Log(Error, fmt.Errorf("Starting command '%s': %v", fullPath, err))
		errString = fmt.Sprintf("There were errors calling external plugin '%s', you might want to ask an administrator to check the logs", task.name)
		return errString, MechanismFail
	}
	if command != "init" {
		emit(ExternalTaskRan)
	}
	if bot.logger == nil {
		var stdErrBytes []byte
		if stdErrBytes, err = ioutil.ReadAll(stderr); err != nil {
			Log(Error, fmt.Errorf("Reading from stderr for external command '%s': %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external plugin '%s', you might want to ask an administrator to check the logs", task.name)
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
			errString = fmt.Sprintf("There were errors calling external plugin '%s', you might want to ask an administrator to check the logs", task.name)
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
