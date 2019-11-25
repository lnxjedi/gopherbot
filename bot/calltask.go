package bot

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/lnxjedi/gopherbot/robot"
)

type getCfgReturn struct {
	buffptr *[]byte
	err     error
}

func getExtDefCfg(task *BotTask) (*[]byte, error) {
	cc := make(chan getCfgReturn)
	go getExtDefCfgThread(cc, task)
	ret := <-cc
	return ret.buffptr, ret.err
}

func getExtDefCfgThread(cchan chan<- getCfgReturn, task *BotTask) {
	var taskPath string
	var err error
	var relpath bool
	if taskPath, err = getTaskPath(task); err != nil {
		cchan <- getCfgReturn{nil, err}
		return
	}
	var cfg []byte
	var cmd *exec.Cmd

	// drop privileges when running external task; this thread will terminate
	// when this goroutine finishes; see runtime.LockOSThread()
	DropThreadPriv(fmt.Sprintf("task %s default configuration", task.name))

	Log(robot.Debug, "Calling '%s' with arg: configure", taskPath)
	//cfg, err = exec.Command(taskPath, "configure").Output()
	cmd = exec.Command(taskPath, "configure")
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
		cchan <- getCfgReturn{nil, err}
		return
	}
	cchan <- getCfgReturn{&cfg, nil}
	return
}

type taskReturn struct {
	errString string
	retval    robot.TaskRetVal
}

// callTask does the work of running a job, task or plugin with a command and arguments.
func (c *botContext) callTask(t interface{}, command string, args ...string) (errString string, retval robot.TaskRetVal) {
	rc := make(chan taskReturn)
	go c.callTaskThread(rc, t, command, args...)
	ret := <-rc
	return ret.errString, ret.retval
}

func (c *botContext) callTaskThread(rchan chan<- taskReturn, t interface{}, command string, args ...string) {
	var errString string
	var retval robot.TaskRetVal

	c.currentTask = t
	r := c.makeRobot()
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	// This should only happen in the rare case that a configured authorizer or elevator is disabled
	if task.Disabled {
		msg := fmt.Sprintf("callTask failed on disabled task %s; reason: %s", task.name, task.reason)
		Log(robot.Error, msg)
		c.debug(msg, false)
		rchan <- taskReturn{msg, robot.ConfigurationError}
		return
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
		if c.directMsg {
			defer checkPanic(r, fmt.Sprintf("Plugin: %s, command: %s, arguments: (omitted)", task.name, command))
		} else {
			defer checkPanic(r, fmt.Sprintf("Plugin: %s, command: %s, arguments: %v", task.name, command, args))
		}
	}
	if c.directMsg {
		Log(robot.Debug, "Dispatching command '%s' to task '%s' with arguments '(omitted for DM)'", command, task.name)
	} else {
		Log(robot.Debug, "Dispatching command '%s' to task '%s' with arguments '%#v'", command, task.name, args)
	}

	// Set up the per-task environment
	envhash := c.getEnvironment(task)

	if isPlugin && plugin.taskType == taskGo {
		if command != "init" {
			emit(GoPluginRan)
		}
		Log(robot.Debug, "Call go plugin: '%s' with args: %q", task.name, args)
		c.taskenvironment = envhash
		ret := pluginHandlers[task.name].Handler(r, command, args...)
		c.taskenvironment = nil
		rchan <- taskReturn{"", ret}
		return
	}
	var taskPath string // full path to the executable
	var err error
	taskPath, err = getTaskPath(task)
	if err != nil {
		emit(ExternalTaskBadPath)
		rchan <- taskReturn{fmt.Sprintf("Error getting path for %s: %v", task.name, err), robot.MechanismFail}
		return
	}
	var externalArgs []string
	// jobs and tasks don't take a 'command' (it's just 'run', a dummy value)
	if isPlugin {
		externalArgs = append(externalArgs, command)
	}
	externalArgs = append(externalArgs, args...)
	Log(robot.Debug, "Calling '%s' with args: %q", taskPath, externalArgs)
	cmd := exec.Command(taskPath, externalArgs...)
	c.Lock()
	c.taskName = task.name
	c.taskDesc = task.Description
	c.osCmd = cmd
	c.Unlock()

	env := make([]string, 0, len(envhash))
	keys := make([]string, 0, len(envhash))
	for k, v := range envhash {
		if len(k) == 0 {
			Log(robot.Error, "Empty Name value while populating environment for '%s', skipping", task.name)
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
	Log(robot.Debug, "Running '%s' in '%s' with environment vars: '%s'", taskPath, cmd.Dir, strings.Join(keys, "', '"))
	var stderr, stdout io.ReadCloser
	// hold on to stderr in case we need to log an error
	stderr, err = cmd.StderrPipe()
	if err != nil {
		Log(robot.Error, "Creating stderr pipe for external command '%s': %v", taskPath, err)
		errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
		rchan <- taskReturn{errString, robot.MechanismFail}
		return
	}
	if c.logger == nil {
		// close stdout on the external plugin...
		cmd.Stdout = nil
	} else {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			Log(robot.Error, "Creating stdout pipe for external command '%s': %v", taskPath, err)
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			rchan <- taskReturn{errString, robot.MechanismFail}
			return
		}
	}

	// drop privileges when running external task; this thread will terminate
	// when this goroutine finishes; see runtime.LockOSThread()
	DropThreadPriv(fmt.Sprintf("task %s / %s", task.name, command))

	if err = cmd.Start(); err != nil {
		Log(robot.Error, "Starting command '%s': %v", taskPath, err)
		errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
		rchan <- taskReturn{errString, robot.MechanismFail}
		return
	}
	if command != "init" {
		emit(ExternalTaskRan)
	}
	if c.logger == nil {
		var stdErrBytes []byte
		if stdErrBytes, err = ioutil.ReadAll(stderr); err != nil {
			Log(robot.Error, "Reading from stderr for external command '%s': %v", taskPath, err)
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			rchan <- taskReturn{errString, robot.MechanismFail}
			return
		}
		stdErrString := string(stdErrBytes)
		if len(stdErrString) > 0 {
			Log(robot.Warn, "Output from stderr of external command '%s': %s", taskPath, stdErrString)
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
		retval = robot.Fail
		success := false
		if exitstatus, ok := err.(*exec.ExitError); ok {
			if status, ok := exitstatus.Sys().(syscall.WaitStatus); ok {
				retval = robot.TaskRetVal(status.ExitStatus())
				if retval == robot.Success {
					success = true
				}
			}
		}
		if !success {
			Log(robot.Error, "Waiting on external command '%s': %v", taskPath, err)
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			emit(ExternalTaskErrExit)
		}
	}
	rchan <- taskReturn{errString, retval}
}
