// +build linux

package bot

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func privThread(reason string) {
	// if privSep {
	// 	runtime.LockOSThread()
	// 	var ruid, euid, suid, nruid, neuid, nsuid uintptr
	// 	syscall.Syscall(syscall.SYS_GETRESUID, uintptr(unsafe.Pointer(&ruid)), uintptr(unsafe.Pointer(&euid)), uintptr(unsafe.Pointer(&suid)))
	// 	tid := syscall.Gettid()
	// 	_, _, errno := syscall.Syscall(syscall.SYS_SETRESUID, uintptr(privUID), uintptr(privUID), uintptr(unprivUID))
	// 	syscall.Syscall(syscall.SYS_GETRESUID, uintptr(unsafe.Pointer(&nruid)), uintptr(unsafe.Pointer(&neuid)), uintptr(unsafe.Pointer(&nsuid)))
	// 	if errno != 0 {
	// 		Log(Error, fmt.Sprintf("Privileged setresuid(%d) call failed for '%s': %d; thread %d r/e/suid: %d/%d/%d", privUID, reason, errno, tid, ruid, euid, suid))
	// 	} else {
	// 		Log(Debug, fmt.Sprintf("Locking raised privileges for '%s' in thread %d; old r/e/suid: %d/%d/%d, new r/e/suid: %d/%d/%d", reason, tid, ruid, euid, suid, nruid, neuid, nsuid))
	// 	}
	// }
}

func unprivThread(reason string) {
	if privSep {
		runtime.LockOSThread()
		var ruid, euid, suid, nruid, neuid, nsuid uintptr
		syscall.Syscall(syscall.SYS_GETRESUID, uintptr(unsafe.Pointer(&ruid)), uintptr(unsafe.Pointer(&euid)), uintptr(unsafe.Pointer(&suid)))
		tid := syscall.Gettid()
		// go1.11.5 should fix the bug where locked OS thread state leaks
		_, _, errno := syscall.Syscall(syscall.SYS_SETRESUID, uintptr(unprivUID), uintptr(unprivUID), uintptr(unprivUID))
		// _, _, errno := syscall.Syscall(syscall.SYS_SETRESUID, uintptr(unprivUID), uintptr(unprivUID), uintptr(privUID))
		syscall.Syscall(syscall.SYS_GETRESUID, uintptr(unsafe.Pointer(&nruid)), uintptr(unsafe.Pointer(&neuid)), uintptr(unsafe.Pointer(&nsuid)))
		if errno != 0 {
			Log(Error, fmt.Sprintf("Unprivileged setresuid(%d) call failed for '%s': %d; thread %d r/e/suid: %d/%d/%d", privUID, reason, errno, tid, ruid, euid, suid))
		} else {
			Log(Debug, fmt.Sprintf("Dropping privileges for '%s' in thread %d; old r/e/suid: %d/%d/%d, new r/e/suid: %d/%d/%d", reason, tid, ruid, euid, suid, nruid, neuid, nsuid))
		}
	}
}

func init() {
	uid := syscall.Getuid()
	euid := syscall.Geteuid()
	if uid != euid {
		privUID = euid
		unprivUID = uid
		runtime.LockOSThread()
		syscall.Syscall(syscall.SYS_SETRESUID, uintptr(euid), uintptr(euid), uintptr(uid))
		privSep = true
	}
}

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
	if taskPath, relpath, err = getTaskPath(task); err != nil {
		cchan <- getCfgReturn{nil, err}
		return
	}
	var cfg []byte
	var cmd *exec.Cmd

	// drop privileges when running external task; this thread will terminate
	// when this goroutine finishes; see runtime.LockOSThread()
	unprivThread(fmt.Sprintf("task %s default configuration", task.name))

	Log(Debug, fmt.Sprintf("Calling '%s' with arg: configure", taskPath))
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
	retval    TaskRetVal
}

// callTask does the work of running a job, task or plugin with a command and arguments.
func (c *botContext) callTask(t interface{}, command string, args ...string) (errString string, retval TaskRetVal) {
	rc := make(chan taskReturn)
	go c.callTaskThread(rc, t, command, args...)
	ret := <-rc
	return ret.errString, ret.retval
}

func (c *botContext) callTaskThread(rchan chan<- taskReturn, t interface{}, command string, args ...string) {
	var errString string
	var retval TaskRetVal

	c.currentTask = t
	r := c.makeRobot()
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	// This should only happen in the rare case that a configured authorizer or elevator is disabled
	if task.Disabled {
		msg := fmt.Sprintf("callTask failed on disabled task %s; reason: %s", task.name, task.reason)
		Log(Error, msg)
		c.debug(msg, false)
		rchan <- taskReturn{msg, ConfigurationError}
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
		Log(Debug, fmt.Sprintf("Dispatching command '%s' to task '%s' with arguments '(omitted for DM)'", command, task.name))
	} else {
		Log(Debug, fmt.Sprintf("Dispatching command '%s' to task '%s' with arguments '%#v'", command, task.name, args))
	}

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
		rchan <- taskReturn{"", ret}
		return
	}
	var taskPath string // full path to the executable
	var err error
	var relpath bool
	taskPath, relpath, err = getTaskPath(task)
	if err != nil {
		emit(ExternalTaskBadPath)
		rchan <- taskReturn{fmt.Sprintf("Error getting path for %s: %v", task.name, err), MechanismFail}
		return
	}
	interpreter, iargs, err := getInterpreter(taskPath, relpath)
	if err != nil {
		errString = "There was a problem calling an external plugin"
		emit(ExternalTaskBadInterpreter)
		rchan <- taskReturn{errString, MechanismFail}
		return
	}
	if len(interpreter) == 0 {
		interpreter = "(none)"
	}
	var externalArgs []string
	if relpath {
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
	Log(Debug, fmt.Sprintf("Calling '%s' with interpreter '%s' and args: %q", taskPath, interpreter, externalArgs))
	var cmd *exec.Cmd
	if relpath {
		// Feed the script to stdin
		cmd = exec.Command(interpreter, externalArgs...)
		script, err := os.Open(filepath.Join(configPath, taskPath))
		if err != nil {
			errString = fmt.Sprintf("opening task '%s': '%v'", taskPath, err)
			emit(ExternalTaskBadPath)
			rchan <- taskReturn{errString, MechanismFail}
			return
		}
		cmd.Stdin = script
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
	envhash["GOPHER_TASK_NAME"] = c.taskName
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
		rchan <- taskReturn{errString, MechanismFail}
		return
	}
	if c.logger == nil {
		// close stdout on the external plugin...
		cmd.Stdout = nil
	} else {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			Log(Error, fmt.Errorf("Creating stdout pipe for external command '%s': %v", taskPath, err))
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			rchan <- taskReturn{errString, MechanismFail}
			return
		}
	}

	// drop privileges when running external task; this thread will terminate
	// when this goroutine finishes; see runtime.LockOSThread()
	unprivThread(fmt.Sprintf("task %s / %s", task.name, command))

	if err = cmd.Start(); err != nil {
		Log(Error, fmt.Errorf("Starting command '%s': %v", taskPath, err))
		errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
		rchan <- taskReturn{errString, MechanismFail}
		return
	}
	if command != "init" {
		emit(ExternalTaskRan)
	}
	if c.logger == nil {
		var stdErrBytes []byte
		if stdErrBytes, err = ioutil.ReadAll(stderr); err != nil {
			Log(Error, fmt.Errorf("Reading from stderr for external command '%s': %v", taskPath, err))
			errString = fmt.Sprintf("There were errors calling external task '%s', you might want to ask an administrator to check the logs", task.name)
			rchan <- taskReturn{errString, MechanismFail}
			return
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
	rchan <- taskReturn{errString, retval}
}
