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
	"syscall"

	"github.com/lnxjedi/gopherbot/robot"
	yaegi "github.com/lnxjedi/gopherbot/v2/modules/yaegi-dynamic-go"
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

func getDefCfg(t interface{}) (*[]byte, error) {
	cc := make(chan getCfgReturn)
	go getDefCfgThread(cc, t)
	ret := <-cc
	return ret.buffptr, ret.err
}

func getDefCfgThread(cchan chan<- getCfgReturn, ti interface{}) {
	var taskPath string
	var err error
	var relpath bool
	var cfg []byte
	var task *Task

	switch t := ti.(type) {
	case *Plugin:
		task = t.Task
		// Reset list of channels
		task.Channels = []string{}
	default:
		log.Panic("getDefCfg called with non-*Plugin interface{}")
	}
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("recovered from panic in getDefCfg for plugin '%s': %v", task.name, p)
			Log(robot.Error, err.Error())
			cchan <- getCfgReturn{&cfg, err}
		}
	}()

	if task.taskType == taskGo {
		plugHandler := pluginHandlers[task.name]
		if plugHandler.Configure != nil {
			defConfig := plugHandler.Configure()
			cchan <- getCfgReturn{defConfig, nil}
			return
		} else {
			cchan <- getCfgReturn{&cfg, nil}
			return
		}
	}

	isExternalGoTask := strings.HasSuffix(task.Path, ".go")
	if taskPath, err = getTaskPath(task, "."); err != nil {
		if !isExternalGoTask && taskPath == "" {
			cchan <- getCfgReturn{nil, err}
			return
		}
		Log(robot.Info, "Calling func Configure for external Go plugin '"+task.name+"'")
		if defConfig, err := yaegi.GetPluginConfig(taskPath, task.name); err != nil {
			Log(robot.Warn, "unable to retrieve plugin default configuration for '%s': %s", task.name, err.Error())
			// This error shouldn't disable an external Go plugin
			cchan <- getCfgReturn{&cfg, nil}
			return
		} else {
			cchan <- getCfgReturn{defConfig, nil}
			return
		}
	}

	var cmd *exec.Cmd

	Log(robot.Debug, "calling '%s' with arg: configure", taskPath)
	cmd = exec.Command(taskPath, "configure")
	if privSep {
		Log(robot.Debug, "PRIVSEP - dropping privileges to uid/gid %d/%d to call configure for '%s'", unprivUID, procGID, task.name)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(unprivUID),
				Gid: uint32(procGID),
			},
		}
	}
	if relpath {
		cmd.Dir = configPath
	}
	env := []string{
		fmt.Sprintf("GOPHER_INSTALLDIR=%s", installPath),
		fmt.Sprintf("RUBYLIB=%s/lib:%s/custom/lib", installPath, homePath),
		fmt.Sprintf("GEM_HOME=%s/.local", homePath),
		fmt.Sprintf("PYTHONPATH=%s/lib:%s/custom/lib", installPath, homePath),
		fmt.Sprintf("GOPHER_CONFIGDIR=%s", configFull),
		fmt.Sprintf("HOME=%s", homePath),
	}
	for _, p := range envPassThrough {
		if value, ok := os.LookupEnv(p); ok {
			env = append(env, fmt.Sprintf("%s=%s", p, value))
		}
	}
	cmd.Env = env
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

// function for active Go Robots to look up the *worker, always locked
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
	task, _, _ := getTask(t)
	go w.callTaskThread(rc, t, command, args...)
	debugPriv(fmt.Sprintf("calltask '%s'", task.name))
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
		if w.Incoming.DirectMessage {
			defer checkPanic(w, fmt.Sprintf("Plugin: %s, command: %s, arguments: (omitted)", task.name, command))
		} else {
			defer checkPanic(w, fmt.Sprintf("Plugin: %s, command: %s, arguments: %v", task.name, command, args))
		}
	}
	if w.Incoming.DirectMessage {
		Log(robot.Debug, "Dispatching command '%s' to task '%s' with arguments '(omitted for DM)'", command, task.name)
	} else {
		Log(robot.Debug, "Dispatching command '%s' to task '%s' with arguments '%#v'", command, task.name, args)
	}

	// Set up the per-task environment, getEnvironment takes lock & releases
	envhash := w.getEnvironment(t)
	r.environment = envhash

	w.registerWorker(r.tid)

	if task.taskType == taskGo {
		defer func() {
			if p := recover(); p != nil {
				err := fmt.Errorf("recovered from panic in callTask for compiled-in Go %s '%s': %v", task.taskType, task.name, p)
				Log(robot.Error, err.Error())
				deregisterWorker(r.tid)
				rchan <- taskReturn{err.Error(), robot.MechanismFail}
			}
		}()
		if isPlugin {

			if command != "init" {
				emit(GoPluginRan)
			}
			Log(robot.Debug, "Calling go plugin: '%s' with args: %q", task.name, args)
			ret := pluginHandlers[task.name].Handler(r, command, args...)
			deregisterWorker(r.tid)
			rchan <- taskReturn{"", ret}
			return
		} else {
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
	isExternalGoTask := strings.HasSuffix(task.Path, ".go")
	var err error
	if task.Homed {
		taskPath, err = getTaskPath(task, ".")
	} else {
		taskPath, err = getTaskPath(task, workdir)
	}
	if err != nil && !isExternalGoTask && taskPath != "" {
		emit(ExternalTaskBadPath)
		rchan <- taskReturn{fmt.Sprintf("Getting the path for %s: %v", task.name, err), robot.MechanismFail}
		return
	}

	if isExternalGoTask {
		if isPlugin {
			if command != "init" {
				emit(GoPluginRan)
			}
			ret, err := yaegi.RunPluginHandler(taskPath, task.name, r, task.Privileged, command, args...)
			if err != nil {
				emit(ExternalTaskBadInterpreter)
				rchan <- taskReturn{fmt.Sprintf("Running plugin %s: %v", task.name, err), robot.MechanismFail}
				return
			}
			deregisterWorker(r.tid)
			rchan <- taskReturn{"", ret}
			return
		} else {
			var ret robot.TaskRetVal
			if isJob {
				ret, err = yaegi.RunJobHandler(taskPath, task.name, r, task.Privileged, args...)
				if err != nil {
					emit(ExternalTaskBadInterpreter)
					rchan <- taskReturn{fmt.Sprintf("Running job %s: %v", task.name, err), robot.MechanismFail}
					return
				}
				r.Log(robot.Debug, "External Go job '%s' executed with args: %q", task.name, args)
			} else {
				ret, err = yaegi.RunTaskHandler(taskPath, task.name, r, task.Privileged, args...)
				if err != nil {
					emit(ExternalTaskBadInterpreter)
					rchan <- taskReturn{fmt.Sprintf("Running task %s: %v", task.name, err), robot.MechanismFail}
					return
				}
				r.Log(robot.Debug, "External Go task '%s' executed with args: %q", task.name, args)
			}
			deregisterWorker(r.tid)
			rchan <- taskReturn{"", ret}
			return
		}
	}

	var externalArgs []string
	// jobs and tasks don't take a 'command' (it's just 'run', a dummy value)
	if isPlugin {
		externalArgs = append(externalArgs, command)
	}
	externalArgs = append(externalArgs, args...)
	Log(robot.Debug, "Calling '%s' with args: %q", taskPath, externalArgs)
	cmd := exec.Command(taskPath, externalArgs...)

	if privSep {
		if privileged {
			Log(robot.Debug, "PRIVSEP - raising privileges to uid/gid/sgids %d/%d/%v to run privileged external task '%s'", privUID, procGID, extraGroups, task.name)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Credential: &syscall.Credential{
					Uid: uint32(privUID),
					Gid: uint32(procGID),
				},
			}
		} else {
			Log(robot.Debug, "PRIVSEP - dropping privileges to uid/gid %d/%d to run external task '%s'", unprivUID, procGID, task.name)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Credential: &syscall.Credential{
					Uid: uint32(unprivUID),
					Gid: uint32(procGID),
				},
			}
		}
	}

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
