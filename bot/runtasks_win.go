// +build windows

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
)

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
		for idx := range args {
			args[idx] = strings.Replace(args[idx], " ", "` ", -1)
			args[idx] = strings.Replace(args[idx], ",", "`,", -1)
			args[idx] = strings.Replace(args[idx], ";", "`;", -1)
			if args[idx] == "" {
				args[idx] = "''"
			}
		}
	}
	return args
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
	var interpreter string
	var iargs []string
	interpreter, iargs, err = getInterpreter(taskPath, relpath)
	if err != nil {
		err = fmt.Errorf("looking up interpreter for %s: %s", taskPath, err)
		return nil, err
	}
	args := fixInterpreterArgs(interpreter, []string{taskPath, "configure"})
	args = append(iargs, args...)
	Log(Debug, fmt.Sprintf("Calling '%s' with args: %q", interpreter, args))
	cmd = exec.Command(interpreter, args...)
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
			desc = fmt.Sprintf("Starting task '%s': %s", task.name, task.Description)
		} else {
			desc = fmt.Sprintf("Starting task '%s'", task.name)
		}
		c.logger.Section(taskinfo, desc)
	}

	if !(task.name == "builtin-admin" && command == "abort") {
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
	interpreter, iargs, err := getInterpreter(taskPath, relpath)
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
	var externalArgs []string
	// on Windows, we exec the interpreter with the script as first arg
	if winInterpreter {
		externalArgs = append(externalArgs, taskPath)
	} else if relpath {
		// When the task path is relative, the script is run from stdin;
		// this allows the script to be run from a different directory
		// than configPath.
		externalArgs = append(externalArgs, "/dev/stdin")
		externalArgs = append(iargs, externalArgs...)
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
		if relpath {
			// Feed the script to stdin
			cmd = exec.Command(interpreter, externalArgs...)
			script, err := os.Open(filepath.Join(configPath, taskPath))
			if err != nil {
				errString = fmt.Sprintf("opening task '%s': '%v'", taskPath, err)
				emit(ExternalTaskBadPath)
				return errString, MechanismFail
			}
			cmd.Stdin = script
		} else {
			cmd = exec.Command(taskPath, externalArgs...)
		}
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
	if filepath.IsAbs(c.workingDirectory) {
		cmd.Dir = c.workingDirectory
	} else {
		if c.protected {
			cmd.Dir = filepath.Join(configPath, c.workingDirectory)
		} else {
			botCfg.RLock()
			cmd.Dir = filepath.Join(botCfg.workSpace, c.workingDirectory)
			botCfg.RUnlock()
		}
	}
	Log(Debug, fmt.Sprintf("Running '%s' in '%s' with environment vars: '%s'", taskPath, cmd.Dir, strings.Join(keys, "', '")))
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
