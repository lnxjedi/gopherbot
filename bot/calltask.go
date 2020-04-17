package bot

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/lnxjedi/robot"
	"golang.org/x/sys/unix"
)

// Set for the terminal connector
var localTerm bool

// Set for the null connector
var nullConn bool

type getCfgReturn struct {
	buffptr *[]byte
	err     error
}

func getExtDefCfg(task *Task) (*[]byte, error) {
	cc := make(chan getCfgReturn)
	go getExtDefCfgThread(cc, task)
	ret := <-cc
	return ret.buffptr, ret.err
}

func getExtDefCfgThread(cchan chan<- getCfgReturn, task *Task) {
	var taskPath string
	var err error
	var relpath bool
	if taskPath, err = getTaskPath(task, "."); err != nil {
		cchan <- getCfgReturn{nil, err}
		return
	}
	var cfg []byte
	var cmd *exec.Cmd

	// drop privileges when running external task; this thread will terminate
	// when this goroutine finishes; see runtime.LockOSThread()
	dropThreadPriv(fmt.Sprintf("task %s default configuration", task.name))

	Log(robot.Debug, "Calling '%s' with arg: configure", taskPath)
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

// Maps populated by callTaskThread, so external tasks can get their Robot
// from the eid (GOPHER_CALLER_ID), and Go tasks can get a handle to the
// *worker from an incrementing tid (task id).
var taskLookup = struct {
	e map[string]Robot
	i map[int]*worker
	sync.RWMutex
}{
	make(map[string]Robot),
	make(map[int]*worker),
	sync.RWMutex{},
}

// register a worker for a tid so Go tasks can look up the *worker
func (w *worker) registerWorker(tid int) {
	taskLookup.Lock()
	taskLookup.i[tid] = w
	taskLookup.Unlock()
}

// deregister the worker when done
func deregisterWorker(tid int) {
	taskLookup.Lock()
	delete(taskLookup.i, tid)
	taskLookup.Unlock()
}

// funtion for active Go Robots to look up the *worker, always locked
// before returning. Note that we always pass a Robot.tid instead of making
// this a method on the Robot, since copying the whole robot for a single
// int is senseless.
func getLockedWorker(idx int) *worker {
	if idx == 0 { // illegal value
		_, file, line, _ := runtime.Caller(1)
		Log(robot.Error, "Illegal call to getLockedWorker with tid = 0 in '%s', line %d", file, line)
		return nil
	}
	taskLookup.RLock()
	w, ok := taskLookup.i[idx]
	taskLookup.RUnlock()
	if !ok {
		_, file, line, _ := runtime.Caller(2)
		Log(robot.Error, "Illegal call to getLockedWorker for inactive worker in '%s', line %d", file, line)
		return nil
	}
	w.Lock()
	return w
}

// callTask does the work of running a job, task or plugin with a command and
// arguments. Note that callTask(Thread) has to concern itself with locking of
// the worker because it can be called within a task by the Elevate() method.
func (w *worker) callTask(t interface{}, command string, args ...string) (errString string, retval robot.TaskRetVal) {
	rc := make(chan taskReturn)
	go w.callTaskThread(rc, t, command, args...)
	ret := <-rc
	return ret.errString, ret.retval
}

func (w *worker) callTaskThread(rchan chan<- taskReturn, t interface{}, command string, args ...string) {
	var errString string
	var retval robot.TaskRetVal
	task, plugin, job := getTask(t)
	isPlugin := plugin != nil
	isJob := job != nil
	w.Lock()
	w.currentTask = t
	logger := w.logger
	workdir := w.workingDirectory
	eid := w.eid
	privileged := w.privileged
	w.Unlock()
	r := w.makeRobot()
	// This should only happen in the rare case that a configured authorizer or elevator is disabled
	if task.Disabled {
		msg := fmt.Sprintf("callTask failed on disabled task %s; reason: %s", task.name, task.reason)
		Log(robot.Error, msg)
		debugT(t, msg, false)
		rchan <- taskReturn{msg, robot.ConfigurationError}
		return
	}
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
	w.section(taskinfo, desc)

	if !(task.name == "builtin-admin" && command == "abort") {
		if w.directMsg {
			defer checkPanic(w, fmt.Sprintf("Plugin: %s, command: %s, arguments: (omitted)", task.name, command))
		} else {
			defer checkPanic(w, fmt.Sprintf("Plugin: %s, command: %s, arguments: %v", task.name, command, args))
		}
	}
	if w.directMsg {
		Log(robot.Debug, "Dispatching command '%s' to task '%s' with arguments '(omitted for DM)'", command, task.name)
	} else {
		Log(robot.Debug, "Dispatching command '%s' to task '%s' with arguments '%#v'", command, task.name, args)
	}

	// Set up the per-task environment, getEnvironment takes lock & releases
	envhash := w.getEnvironment(t)
	r.environment = envhash

	w.registerWorker(r.tid)
	if isPlugin && plugin.taskType == taskGo {
		if command != "init" {
			emit(GoPluginRan)
		}
		Log(robot.Debug, "Calling go plugin: '%s' with args: %q", task.name, args)
		ret := pluginHandlers[task.name].Handler(r, command, args...)
		deregisterWorker(r.tid)
		rchan <- taskReturn{"", ret}
		return
	} else if task.taskType == taskGo {
		Log(robot.Debug, "Calling go task '%s' (type %s) with args: %q", task.name, task.taskType, args)
		var ret robot.TaskRetVal
		if isJob {
			ret = jobHandlers[task.name].Handler(r, args...)
		} else {
			ret = taskHandlers[task.name].Handler(r, args...)
		}
		deregisterWorker(r.tid)
		rchan <- taskReturn{"", ret}
		return
	}

	// Task lookup; add lookup for http.go
	taskLookup.Lock()
	taskLookup.e[eid] = r
	taskLookup.Unlock()
	defer func() {
		taskLookup.Lock()
		delete(taskLookup.e, eid)
		taskLookup.Unlock()
		deregisterWorker(r.tid)
	}()

	var taskPath string // full path to the executable
	var err error
	if task.Homed {
		taskPath, err = getTaskPath(task, ".")
	} else {
		taskPath, err = getTaskPath(task, workdir)
	}
	if err != nil {
		emit(ExternalTaskBadPath)
		rchan <- taskReturn{fmt.Sprintf("Getting path for %s: %v", task.name, err), robot.MechanismFail}
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

	// Homed tasks ALWAYS run in cwd, Homed pipelines may have modified the
	// working directory with SetWorkingDirectory.
	if task.Homed {
		cmd.Dir = "."
	} else {
		cmd.Dir = workdir
	}
	if task.Privileged || task.Homed {
		if task.Privileged && len(homePath) > 0 {
			// May already be provided for a privileged pipeline
			envhash["GOPHER_HOME"] = homePath
		}
		// Always set for homed and privileged tasks
		envhash["GOPHER_WORKSPACE"] = r.cfg.workSpace
		envhash["GOPHER_CONFIGDIR"] = configFull
	}
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
	Log(robot.Debug, "Running '%s' in '%s' with environment vars: '%s'", taskPath, cmd.Dir, strings.Join(keys, "', '"))
	var stderr, stdout io.ReadCloser
	// hold on to stderr in case we need to log an error
	stderr, err = cmd.StderrPipe()
	if err != nil {
		Log(robot.Error, "Creating stderr pipe for external command '%s': %v", taskPath, err)
		errString = fmt.Sprintf("Pipeline failed in external task '%s', writing fail log in GOPHER_HOME", task.name)
		rchan <- taskReturn{errString, robot.MechanismFail}
		return
	}
	// Null connector can read from stdin
	if nullConn {
		cmd.Stdin = os.Stdin
	}
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		Log(robot.Error, "Creating stdout pipe for external command '%s': %v", taskPath, err)
		errString = fmt.Sprintf("Pipeline failed in external task '%s', writing fail log in GOPHER_HOME", task.name)
		rchan <- taskReturn{errString, robot.MechanismFail}
		return
	}

	if privileged {
		if isPlugin && !plugin.Privileged {
			dropThreadPriv(fmt.Sprintf("task %s / %s", task.name, command))
		} else {
			raiseThreadPrivExternal(fmt.Sprintf("task %s / %s", task.name, command))
		}
	} else {
		dropThreadPriv(fmt.Sprintf("task %s / %s", task.name, command))
	}

	// Create separate process group to enable killing the process group
	cmd.SysProcAttr = &unix.SysProcAttr{Setpgid: true}
	if err = cmd.Start(); err != nil {
		Log(robot.Error, "Starting command '%s': %v", taskPath, err)
		errString = fmt.Sprintf("Pipeline failed in external task '%s', writing fail log in GOPHER_HOME", task.name)
		rchan <- taskReturn{errString, robot.MechanismFail}
		return
	}
	w.Lock()
	w.osCmd = cmd
	w.Unlock()
	defer func() {
		w.Lock()
		w.osCmd = nil
		w.Unlock()
	}()
	if command != "init" {
		emit(ExternalTaskRan)
	}
	closed := make(chan struct{})
	var solog, selog *log.Logger
	if localTerm {
		solog = log.New(terminalWriter, "OUT: ", 0)
		selog = log.New(terminalWriter, "ERR: ", 0)
	}
	if nullConn {
		solog = log.New(os.Stdout, "", 0)
		selog = log.New(os.Stderr, "ERR: ", 0)
	}
	go func() {
		logging := logger != nil
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if logging {
				logger.Log("OUT " + line)
			}
			if localTerm || nullConn {
				solog.Println(line)
			}
		}
		closed <- struct{}{}
	}()
	go func() {
		logging := logger != nil
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if logging {
				logger.Log("ERR " + line)
			}
			if localTerm || nullConn {
				selog.Println(line)
			}
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
	if err = cmd.Wait(); err != nil {
		retval = robot.Fail
		success := false
		if exitstatus, ok := err.(*exec.ExitError); ok {
			if status, ok := exitstatus.Sys().(unix.WaitStatus); ok {
				retval = robot.TaskRetVal(status.ExitStatus())
				if retval == robot.Success {
					success = true
				}
			}
		}
		if !success {
			Log(robot.Error, "Waiting on external command '%s': %v", taskPath, err)
			errString = fmt.Sprintf("Pipeline failed in external task '%s', writing fail log in GOPHER_HOME", task.name)
			emit(ExternalTaskErrExit)
		}
	}
	rchan <- taskReturn{errString, retval}
}
