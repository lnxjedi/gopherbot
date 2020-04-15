package bot

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/robot"
)

/* robot_methods.go defines some convenience functions on struct Robot to
   simplify use by plugins. */

// Robot is the internal struct for a robot.Message, with bits copied
// from the pipeContext; see that struct for better descriptions.
// A new Robot is created for every task, plugin or job executed by
// callTask(...).
type Robot struct {
	*robot.Message
	tid          int                         // task ID for looking up the *worker
	pipeContext                              // snapshot copy of pipeline context
	cfg          *configuration              // convenience only; r.cfg shorter than r.worker.cfg
	tasks        *taskList                   // same
	maps         *userChanMaps               // same
	repositories map[string]robot.Repository // same
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
		pipeContext: pipeContext{
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
			nsExtension:    w.nsExtension,
			currentTask:    w.currentTask,
			exclusive:      w.exclusive,
		},
		cfg:          w.cfg,
		tasks:        w.tasks,
		maps:         w.maps,
		repositories: w.repositories,
	}
	return r
}

// CheckAdmin returns true if the user is a configured administrator of the
// robot, and true for automatic tasks. Should be used sparingly, when a single
// plugin has multiple commands, some which require admin. Otherwise the plugin
// should just configure RequireAdmin: true
func (r Robot) CheckAdmin() bool {
	// Note that this does "the right thing", using the user from the worker;
	// the user in the Robot is writeable.
	w := getLockedWorker(r.tid)
	w.Unlock()
	return w.CheckAdmin()
}

func (w *worker) CheckAdmin() bool {
	if w.automaticTask {
		return true
	}
	for _, adminUser := range w.cfg.adminUsers {
		if w.User == adminUser {
			emit(AdminCheckPassed)
			return true
		}
	}
	emit(AdminCheckFailed)
	return false
}

// RaisePriv lets go plugins raise privilege for a thread, allowing filesystem
// access in GOPHER_HOME.
func (r Robot) RaisePriv(reason string) {
	raiseThreadPriv(reason)
}

// SetParameter sets a parameter for the current pipeline, useful only for
// passing parameters (as environment variables) to tasks later in the pipeline.
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

// SetWorkingDirectory sets the working directory of the pipeline for all scripts
// executed. The value of path is interpreted as follows:
// * "/absolute/path" - tasks that follow will start with this workingDirectory;
//   "cleanup" won't work, see tasks/cleanup.sh (unsafe)
// * "relative/path" - sets workingDirectory relative to baseDirectory;
//   workSpace or $(pwd) depending on value of Homed for the job/plugin starting
//   the pipeline
// * "./sub/directory" - appends to the current workingDirectory
// * "." - resets workingDirectory to baseDirectory
// Fails if the new working directory doesn't exist
// See also: tasks/setworkdir.sh for updating working directory in a pipeline
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

// GetParameter retrieves the value of a parameter for a namespace. Only useful
// for Go plugins; external scripts have all parameters for the NameSpace stored
// as environment variables. Note that runtasks.go populates the environment
// with Stored parameters, too. So GetParameter is useful for both short-term
// parameters in a pipeline, and for getting long-term parameters such as
// credentials.
func (r Robot) GetParameter(key string) string {
	value, ok := r.environment[key]
	if ok {
		return value
	}
	return ""
}

// Elevate lets a plugin request elevation on the fly. When immediate = true,
// the elevator should always prompt for 2fa; otherwise a configured timeout
// should apply.
func (r Robot) Elevate(immediate bool) bool {
	task, _, _ := getTask(r.currentTask)
	retval := r.elevate(task, immediate)
	if retval == robot.Success {
		return true
	}
	return false
}

// Fixed is a deprecated convenience function for sending a message with fixed width
// font.
func (r Robot) Fixed() robot.Robot {
	nr := r
	m := *r.Message
	nr.Message = &m
	nr.Format = robot.Fixed
	return nr
}

// MessageFormat returns a robot object with the given format, most likely for a
// plugin that will mostly use e.g. Variable format.
func (r Robot) MessageFormat(f robot.MessageFormat) robot.Robot {
	nr := r
	m := *r.Message
	nr.Message = &m
	nr.Format = f
	return nr
}

// Direct is a convenience function for initiating a DM conversation with a
// user. Created initially so a plugin could prompt for a password in a DM.
func (r Robot) Direct() robot.Robot {
	nr := r
	m := *r.Message
	nr.Message = &m
	nr.Channel = ""
	return nr
}

// Pause is a convenience function to pause some fractional number of seconds.
func (r Robot) Pause(s float64) {
	ms := time.Duration(s * float64(1000))
	time.Sleep(ms * time.Millisecond)
}

// RandomString is a convenience function for returning a random string
// from a slice of strings, so that replies can vary.
func (r Robot) RandomString(s []string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	return s[random.Intn(l)]
}

// RandomInt uses the robot's seeded random to return a random int 0 <= retval < n
func (r Robot) RandomInt(n int) int {
	return random.Intn(n)
}

// GetBotAttribute returns an attribute of the robot or "" if unknown.
// Current attributes:
// name, alias, fullName, contact
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
	default:
		ret = robot.AttributeNotFound
	}
	return &robot.AttrRet{attr, ret}
}

/*

GetTaskConfig sets a struct pointer to point to a config struct populated
from configuration when plugins were loaded. To use, a plugin should define
a struct for it's configuration data, e.g.:

	type pConf struct {
		Username, Password string
	}

In conf/plugins/<pluginname>.yaml, you would add a Config: stanza, e.g.:

	Config:
	  Username: foo
	  Password: bar

When registering the plugin, you pass a pointer to an empty config template, which the
robot will use to populate a struct when configuration is loaded:

	func init() {
		bot.RegisterPlugin("memes", bot.PluginHandler{
			DefaultConfig: defaultConfig, // yaml string providing default configuration
			Handler:       plugfunc, // callback function
			Config:        &pConf{}, // pointer to empty config struct
		})
	}

Then, to get a current copy of configuration when the plugin runs, define a struct pointer
and call GetTaskConfig with a double-pointer:

	var c *pConf
	r.GetTaskConfig(&c)

... And voila! *pConf is populated with the contents from the configured Config: stanza
*/
func (r Robot) GetTaskConfig(dptr interface{}) robot.RetVal {
	task, _, _ := getTask(r.currentTask)
	if task.config == nil {
		Log(robot.Debug, "Task \"%s\" called GetTaskConfig, but no config was found.", task.name)
		return robot.NoConfigFound
	}
	tp := reflect.ValueOf(dptr)
	if tp.Kind() != reflect.Ptr {
		Log(robot.Debug, "Task \"%s\" called GetTaskConfig, but didn't pass a double-pointer to a struct", task.name)
		return robot.InvalidDblPtr
	}
	p := reflect.Indirect(tp)
	if p.Kind() != reflect.Ptr {
		Log(robot.Debug, "Task \"%s\" called GetTaskConfig, but didn't pass a double-pointer to a struct", task.name)
		return robot.InvalidDblPtr
	}
	if p.Type() != reflect.ValueOf(task.config).Type() {
		Log(robot.Debug, "Task \"%s\" called GetTaskConfig with an invalid double-pointer", task.name)
		return robot.InvalidCfgStruct
	}
	p.Set(reflect.ValueOf(task.config))
	return robot.Ok
}

// Log logs a message to the robot's log file (or stderr) if the level
// is lower than or equal to the robot's current log level
func (r Robot) Log(l robot.LogLevel, msg string, v ...interface{}) (logged bool) {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	if Log(l, msg) && r.logger != nil {
		line := "LOG " + logLevelToStr(l) + " " + msg
		r.logger.Log(strings.TrimSpace(line))
	}
	return
}
