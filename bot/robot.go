package bot

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

/* robot_methods.go defines some convenience functions on struct Robot to
   simplify use by plugins. */

// Robot is the internal struct for a robot.Message, with bits copied
// from the pipeContext; see that struct for better descriptions.
// A new Robot is created for every task, plugin or job executed by
// callTask(...).
type Robot struct {
	*robot.Message
	tid          int            // task ID for looking up the *worker
	*pipeContext                // snapshot copy of pipeline context
	cfg          *configuration // convenience only; r.cfg shorter than r.worker.cfg
	tasks        *taskList      // same
	maps         *userChanMaps  // same
}

// Incrementing tid for individual tasks that run, so Go Robots
// can look up the *worker when needed.
var taskID = struct {
	idx int
	sync.Mutex
}{
	0,
	sync.Mutex{},
}

// Get the next task ID; 0 is an illegal value
func getTaskID() int {
	taskID.Lock()
	taskID.idx++
	if taskID.idx == 0 {
		taskID.idx = 1
	}
	tid := taskID.idx
	taskID.Unlock()
	return tid
}

// makeRobot returns a Robot for plugins; the tid lets Robot methods
// get a reference back to the original context when needed. The Robot
// should contain a copy of almost all of the information needed for plugins
// to run.
func (w *worker) makeRobot() Robot {
	r := Robot{
		tid: getTaskID(),
		// Copy these bits, which can be modified for an individual Robot
		Message: &robot.Message{
			User:            w.User,
			ProtocolUser:    w.ProtocolUser,
			Channel:         w.Channel,
			ProtocolChannel: w.ProtocolChannel,
			Format:          w.Format,
			Protocol:        w.Protocol,
			Incoming:        w.Incoming,
		},
		cfg:   w.cfg,
		tasks: w.tasks,
		maps:  w.maps,
	}
	if w.pipeContext != nil {
		r.pipeContext = &pipeContext{
			privileged:     w.privileged,
			timeZone:       w.timeZone,
			logger:         w.logger,
			ptype:          w.ptype,
			elevated:       w.elevated,
			stage:          w.stage,
			jobInitialized: w.jobInitialized,
			jobName:        w.jobName,
			nameSpace:      w.nameSpace,
			pipeName:       w.pipeName,
			pipeDesc:       w.pipeDesc,
			currentTask:    w.currentTask,
			exclusive:      w.exclusive,
		}
	}
	return r
}

func (w *worker) makeMemoryContext(key string) memoryContext {
	var threadID string
	if w.Incoming.ThreadedMessage {
		threadID = w.Incoming.ThreadID
	}
	return memoryContext{
		key:     key,
		user:    w.Incoming.UserID,
		channel: w.Channel,
		thread:  threadID,
	}
}

func (r Robot) makeMemoryContext(key string, forceThread, shared bool) memoryContext {
	var threadID string
	if r.Incoming.ThreadedMessage || forceThread {
		threadID = r.Incoming.ThreadID
	}
	user := r.Incoming.UserID
	// if len(r.Channel) == 0, it's a direct message to the robot
	// and the idea of shared is meaningless - plus we NEED the user ID
	// to differentiate between different user's DM memories
	if shared && len(r.Channel) > 0 {
		user = ""
	}
	return memoryContext{
		key:     key,
		user:    user,
		channel: r.Channel,
		thread:  threadID,
	}
}

// see robot/robot.go
func (r Robot) CheckAdmin() bool {
	// Note that this does "the right thing", using the user from the worker;
	// the user in the Robot is writeable.
	w := getLockedWorker(r.tid)
	w.Unlock()
	return w.checkAdmin()
}

func (w *worker) checkAdmin() bool {
	if w.automaticTask {
		return true
	}
	for _, adminUser := range w.cfg.adminUsers {
		if w.User == adminUser {
			if !w.listedUser {
				Log(robot.Error, "admin user %s not listed in roster; failing admin check", w.User)
				emit(AdminCheckFailed)
				return false
			}
			emit(AdminCheckPassed)
			return true
		}
	}
	emit(AdminCheckFailed)
	return false
}

// see robot/robot.go
func (r Robot) RaisePriv(reason string) {
	raiseThreadPriv(reason)
}

// see robot/robot.go
func (r Robot) SetParameter(name, value string) bool {
	if !identifierRe.MatchString(name) {
		return false
	}
	w := getLockedWorker(r.tid)
	defer w.Unlock()
	c := w.pipeContext
	c.environment[name] = value
	return true
}

// see robot/robot.go
func (r Robot) SetWorkingDirectory(path string) bool {
	w := getLockedWorker(r.tid)
	defer w.Unlock()
	c := w.pipeContext
	if path == "." {
		c.workingDirectory = c.baseDirectory
		return true
	}
	if filepath.IsAbs(path) {
		raiseThreadPriv("checking absolute path")
		_, ok := checkDirectory(path)
		if ok {
			c.workingDirectory = path
		} else {
			r.Log(robot.Error, "Invalid path '%s' in SetWorkingDirectory", path)
		}
		return ok
	}
	if strings.HasPrefix(path, "./") {
		checkPath := filepath.Join(c.workingDirectory, path)
		_, ok := checkDirectory(checkPath)
		if ok {
			c.workingDirectory = checkPath
		} else {
			r.Log(robot.Error, "Invalid path '%s'(%s) in SetWorkingDirectory", path, checkPath)
		}
		return ok
	}
	checkPath := filepath.Join(c.baseDirectory, path)
	_, ok := checkDirectory(checkPath)
	if ok {
		c.workingDirectory = checkPath
	} else {
		r.Log(robot.Error, "Invalid path '%s'(%s) in SetWorkingDirectory", path, checkPath)
	}
	return ok
}

// see robot/robot.go
func (r Robot) GetParameter(key string) string {
	value, ok := r.environment[key]
	if ok {
		return value
	}
	return ""
}

// see robot/robot.go
func (r Robot) Elevate(immediate bool) bool {
	task, _, _ := getTask(r.currentTask)
	retval := r.elevate(task, immediate)
	return retval == robot.Success
}

// see robot/robot.go
func (r Robot) Fixed() robot.Robot {
	nr := r
	m := *r.Message
	nr.Message = &m
	nr.Format = robot.Fixed
	return nr
}

// see robot/robot.go
func (r Robot) MessageFormat(f robot.MessageFormat) robot.Robot {
	nr := r
	m := *r.Message
	nr.Message = &m
	nr.Format = f
	return nr
}

// see robot/robot.go
func (r Robot) Direct() robot.Robot {
	nr := r
	m := *r.Message
	nr.Message = &m
	nr.Channel = ""
	return nr
}

// see robot/robot.go
func (r Robot) Threaded() robot.Robot {
	nr := r
	m := *r.Message
	nr.Message = &m
	if len(nr.Channel) > 0 {
		nr.Incoming.ThreadedMessage = true
	} else {
		nr.Incoming.ThreadedMessage = false
	}
	return nr
}

// see robot/robot.go
func (r Robot) Pause(s float64) {
	ms := time.Duration(s * float64(1000))
	time.Sleep(ms * time.Millisecond)
}

// see robot/robot.go
func (r Robot) RandomString(s []string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	return s[random.Intn(l)]
}

// see robot/robot.go
func (r Robot) RandomInt(n int) int {
	return random.Intn(n)
}

// see robot/robot.go
func (r Robot) GetBotAttribute(a string) *robot.AttrRet {
	a = strings.ToLower(a)
	ret := robot.Ok
	var attr string
	switch a {
	case "name":
		attr = r.cfg.botinfo.UserName
	case "fullname", "realname":
		attr = r.cfg.botinfo.FullName
	case "alias":
		attr = string(r.cfg.alias)
	case "mail", "email":
		attr = r.cfg.botinfo.Email
	case "contact", "admin", "admincontact":
		attr = r.cfg.adminContact
	case "protocol":
		attr = r.Protocol.String()
	case "id", "internalid", "protocolid":
		attr = fmt.Sprintf("<%s>", r.cfg.botinfo.UserID)
	default:
		ret = robot.AttributeNotFound
	}
	return &robot.AttrRet{attr, ret}
}

// see robot/robot.go
func (r Robot) GetTaskConfig(dptr interface{}) robot.RetVal {
	task, _, _ := getTask(r.currentTask)
	if task.config == nil {
		Log(robot.Error, "Task \"%s\" called GetTaskConfig, but no config was found.", task.name)
		return robot.NoConfigFound
	}
	tp := reflect.ValueOf(dptr)
	if tp.Kind() != reflect.Ptr {
		Log(robot.Error, "Task \"%s\" called GetTaskConfig, but didn't pass a double-pointer to a struct", task.name)
		return robot.InvalidDblPtr
	}
	p := reflect.Indirect(tp)
	if p.Kind() != reflect.Ptr {
		Log(robot.Error, "Task \"%s\" called GetTaskConfig, but didn't pass a double-pointer to a struct", task.name)
		return robot.InvalidDblPtr
	}
	if p.Type() != reflect.ValueOf(task.config).Type() {
		Log(robot.Error, "Task \"%s\" called GetTaskConfig with an invalid double-pointer", task.name)
		return robot.InvalidCfgStruct
	}
	p.Set(reflect.ValueOf(task.config))
	return robot.Ok
}

// see robot/robot.go
func (r Robot) Log(l robot.LogLevel, msg string, v ...interface{}) (logged bool) {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	logged = Log(l, msg)
	// All robot Log calls get logged to terminal output
	if !logged && localTerm {
		if terminalWriter != nil {
			terminalWriter.Write([]byte("LOG " + logLevelToStr(l) + ": " + msg + "\n"))
		} else {
			botStdOutLogger.Print("LOG " + logLevelToStr(l) + ": " + msg)
		}
	}
	if r.logger != nil {
		line := "LOG " + logLevelToStr(l) + ": " + msg
		r.logger.Log(strings.TrimSpace(line))
	}
	return
}
