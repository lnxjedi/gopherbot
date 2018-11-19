package bot

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// MessageFormat indicates how the connector should display the content of
// the message. One of Variable, Fixed or Raw
type MessageFormat int

// Outgoing message format, Variable or Fixed
const (
	Raw MessageFormat = iota // protocol native, zero value -> default if not specified
	Fixed
	Variable
)

// Connector protocols
type Protocol int

const (
	Slack Protocol = iota
	Terminal
	Test
)

// Robot is passed to each task as it runs, initialized from the botContext.
// Tasks can copy and modify the Robot without affecting the botContext.
type Robot struct {
	User     string        // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel  string        // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Protocol Protocol      // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	RawMsg   interface{}   // raw struct of message sent by connector; interpret based on protocol. For Slack this is a *slack.MessageEvent
	Format   MessageFormat // The outgoing message format, one of Raw, Fixed, or Variable
	id       int           // For looking up the botContext
}

//go:generate stringer -type=Protocol

// Generate String method with: go generate ./bot/

/* robot_methods.go defines some convenience functions on struct Robot to
   simplify use by plugins. */

// getContext returns the botContext for a given Robot
func (r *Robot) getContext() *botContext {
	return getBotContextInt(r.id)
}

// CheckAdmin returns true if the user is a configured administrator of the
// robot, and true for automatic tasks. Should be used sparingly, when a single
// plugin has multiple commands, some which require admin. Otherwise the plugin
// should just configure RequireAdmin: true
func (r *Robot) CheckAdmin() bool {
	c := r.getContext()
	if c.automaticTask {
		return true
	}
	botCfg.RLock()
	defer botCfg.RUnlock()
	for _, adminUser := range botCfg.adminUsers {
		if r.User == adminUser {
			emit(AdminCheckPassed)
			return true
		}
	}
	emit(AdminCheckFailed)
	return false
}

// SetParameter sets a parameter for the current pipeline, useful only for
// passing parameters (as environment variables) to tasks later in the pipeline.
func (r *Robot) SetParameter(name, value string) bool {
	if !identifierRe.MatchString(name) {
		return false
	}
	c := r.getContext()
	c.environment[name] = value
	return true
}

// SetWorkingDirectory sets the working directory of the pipeline for all scripts
// executed. The path argument can be absolute or relative; if relative, it is
// always relative to the robot's WorkSpace.
func (r *Robot) SetWorkingDirectory(path string) bool {
	var newPath string
	if filepath.IsAbs(path) {
		newPath = path
	} else {
		newPath = filepath.Join(botCfg.workSpace, path)
	}
	if respath, ok := checkDirectory(newPath); ok {
		c := r.getContext()
		c.workingDirectory = respath
		return true
	} else {
		r.Log(Error, fmt.Sprintf("Invalid path '%s'(%s) in SetWorkingDirectory", path, newPath))
		return false
	}
}

// ExtendNamespace is for CI/CD applications to support building multiple
// repositories from a single triggered job. When ExtendNamespace is called,
// all future long-term memory lookups are prefixed with the extended
// namespace, and a new history is started for the extended namespace.
// It is an error to call ExtendNamespace twice in a single job pipeline, or
// outside of a running job. The histories argument is interpreted as the
// number of histories to keep for the extended namespace, or -1 to inherit
// from the parent job.
// Arguments:
// ext (extension) => "<repository>/<branch>", where repository is listed in
//   repositories.yaml
// histories => number of histories to keep
func (r *Robot) ExtendNamespace(ext string, histories int) bool {
	if strings.ContainsRune(ext, ':') {
		r.Log(Error, "Invalid namespact extension contains ':'")
		return false
	}
	c := r.getContext()
	if c.stage != primaryTasks {
		r.Log(Error, "ExtendNamespace called after pipeline end")
		return false
	}
	if len(c.jobName) == 0 {
		r.Log(Error, "ExtendNamespace called with no job in progress")
		return false
	}
	if len(c.nsExtension) > 0 {
		r.Log(Error, "ExtendNamespace called after namespace already extended")
		return false
	}
	cmp := strings.Split(ext, "/")
	repo := strings.Join(cmp[0:len(cmp)-1], "/")
	if _, exists := c.repositories[repo]; !exists {
		r.Log(Error, fmt.Sprintf("Repository '%s' not found in repositories.yaml", ext))
		return false
	}
	r.Log(Debug, fmt.Sprintf("Extending namespace for job '%s': %s", c.jobName, ext))
	c.nsExtension = ext

	jk := histPrefix + c.jobName
	var pjh jobHistory
	jtok, _, jret := checkoutDatum(jk, &pjh, true)
	if jret != Ok {
		r.Log(Error, fmt.Sprintf("Problem checking out '%s', unable to record extended namespace '%s'", jk, ext))
	} else {
		xn := make(map[string]bool)
		for _, v := range pjh.ExtendedNamespaces {
			xn[v] = true
		}
		xn[ext] = true
		pjh.ExtendedNamespaces = make([]string, len(xn))
		i := 0
		for k, _ := range xn {
			pjh.ExtendedNamespaces[i] = k
			i++
		}
		ret := updateDatum(jk, jtok, pjh)
		if ret != Ok {
			r.Log(Error, fmt.Sprintf("Problem updating '%s', unable to record extended namespace '%s'", jk, ext))
		}
	}

	var nh int
	if histories != -1 {
		nh = histories
	} else {
		j := c.tasks.getTaskByName(c.jobName)
		_, _, job := getTask(j)
		nh = job.HistoryLogs
	}
	var jh jobHistory
	rememberRuns := nh
	if rememberRuns == 0 {
		rememberRuns = 1
	}
	key := histPrefix + c.jobName + ":" + ext
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
			if nh > 0 && c.history != nil {
				pipeHistory, err := c.history.NewHistory(c.pipeName+":"+ext, hist.LogIndex, nh)
				if err != nil {
					Log(Error, fmt.Sprintf("Error starting history for '%s', no history will be recorded: %v", c.pipeName, err))
				} else {
					if c.logger != nil {
						c.logger.Section("close log", fmt.Sprintf("Job '%s' extended namespace: '%s'; starting new log on next task", c.jobName, ext))
					}
					c.logger = pipeHistory
					c.logger.Section("new log", fmt.Sprintf("Extended log created by job '%s'", c.jobName))
					r.Log(Debug, fmt.Sprintf("Started new history for job '%s' with namespace '%s'", c.jobName, ext))
					if c.verbose {
						r.Channel = c.jobChannel
						r.Say(fmt.Sprintf("Job '%s' extended namespace: %s:%s, run %d", c.jobName, c.jobName, ext, c.runIndex))
					}
				}
			} else {
				if c.history == nil {
					Log(Warn, "Error starting history, no history provider available")
				}
			}
		}
	}
	repository, _ := c.repositories[ext]
	for _, param := range repository.Parameters {
		name := param.Name
		value := param.Value
		_, exists := c.environment[name]
		if !exists {
			c.environment[name] = value
		}
	}
	// Populate the environment with secrets for this repository. Task secrets
	// are populated in runtasks.go/callTask
	cryptKey.RLock()
	initialized := cryptKey.initialized
	ckey := cryptKey.key
	cryptKey.RUnlock()
	if initialized {
		repEnv, exists := c.storedEnv.RepositoryParams[ext]
		if exists {
			if initialized {
				for name, encvalue := range repEnv {
					_, exists := c.environment[name]
					if !exists {
						value, err := decrypt(encvalue, ckey)
						if err != nil {
							Log(Error, fmt.Sprintf("Error decrypting '%s' for repository '%s': %v", name, ext, err))
							break
						}
						c.environment[name] = string(value)
					}
				}
			}
		}
		repEnv, exists = c.storedEnv.RepositoryParams[repo]
		if exists {
			if initialized {
				for name, encvalue := range repEnv {
					_, exists := c.environment[name]
					if !exists {
						value, err := decrypt(encvalue, ckey)
						if err != nil {
							Log(Error, fmt.Sprintf("Error decrypting '%s' for repository '%s': %v", name, ext, err))
							break
						}
						c.environment[name] = string(value)
					}
				}
			}
		}
	}
	return true
}

// SpawnTask creates a new botContext in a new goroutine to run a
// plugin/task/job. It's primary use is for CI/CD applications where a single
// triggered job may want to spawn several jobs when e.g. a dependency for
// multiple projects is updated.
func (r *Robot) SpawnTask(name string, cmdargs ...string) RetVal {
	c := r.getContext()
	if c.stage != primaryTasks {
		task, _, _ := getTask(c.currentTask)
		r.Log(Error, fmt.Sprintf("SpawnTask called outside of initial pipeline in task '%s'", task.name))
		return InvalidStage
	}
	t := c.tasks.getTaskByName(name)
	if t == nil {
		task, _, _ := getTask(c.currentTask)
		r.Log(Error, fmt.Sprintf("Task '%s' not found in call to AddTask from task '%s'", name, task.name))
		return TaskNotFound
	}
	_, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	var command string
	var args []string
	if isPlugin {
		if len(cmdargs) == 0 {
			return MissingArguments
		}
		if len(cmdargs[0]) == 0 {
			return MissingArguments
		}
		command, args = cmdargs[0], cmdargs[1:]
	} else {
		command = "run"
		args = cmdargs
	}
	sb := c.clone()
	go sb.startPipeline(nil, t, spawnedTask, command, args...)
	return Ok
}

// AddTask puts another task (job or plugin) in the queue for the pipeline. Unlike other
// CI/CD tools, gopherbot pipelines are code generated, not configured; it is,
// however, trivial to write code that reads an arbitrary configuration file
// and uses AddTask to generate a pipeline. When the task is a plugin, cmdargs
// should be a command followed by arguments. For jobs, cmdargs are just
// arguments passed to the job.
func (r *Robot) AddTask(name string, cmdargs ...string) RetVal {
	c := r.getContext()
	if c.stage != primaryTasks {
		task, _, _ := getTask(c.currentTask)
		r.Log(Error, fmt.Sprintf("AddTask called outside of initial pipeline in task '%s'", task.name))
		return InvalidStage
	}
	t := c.tasks.getTaskByName(name)
	if t == nil {
		task, _, _ := getTask(c.currentTask)
		r.Log(Error, fmt.Sprintf("Task '%s' not found in call to AddTask from task '%s'", name, task.name))
		return TaskNotFound
	}
	_, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	var command string
	var args []string
	if isPlugin {
		if len(cmdargs) == 0 {
			return MissingArguments
		}
		if len(cmdargs[0]) == 0 {
			return MissingArguments
		}
		command, args = cmdargs[0], cmdargs[1:]
	} else {
		command = "run"
		args = cmdargs
	}
	ts := taskSpec{
		Name:      name,
		Command:   command,
		Arguments: args,
		task:      t,
	}
	c.nextTasks = append(c.nextTasks, ts)
	return Ok
}

// FinalTask adds a task that always runs when the pipeline ends. This
// can be used to ensure that cleanup tasks like terminating a VM or stopping
// the ssh-agent will run, regardless of whether the pipeline failed.
// Note that unlike other tasks, final tasks are run in reverse of the order
// they're added.
func (r *Robot) FinalTask(name string, cmdargs ...string) RetVal {
	c := r.getContext()
	if c.stage != primaryTasks {
		return InvalidStage
	}
	t := c.tasks.getTaskByName(name)
	if t == nil {
		return TaskNotFound
	}
	_, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	var command string
	var args []string
	if isPlugin {
		if len(cmdargs) == 0 {
			return MissingArguments
		}
		if len(cmdargs[0]) == 0 {
			return MissingArguments
		}
		command, args = cmdargs[0], cmdargs[1:]
	} else {
		command = "run"
		args = cmdargs
	}
	ts := taskSpec{
		Name:      name,
		Command:   command,
		Arguments: args,
		task:      t,
	}
	// Final tasks are FILO/LIFO (run in reverse order of being added)
	c.finalTasks = append([]taskSpec{ts}, c.finalTasks...)
	return Ok
}

// FailTask adds a task that runs if the pipeline fails. This can be used to
// e.g. terminate a VM that shouldn't be left running if the pipeline fails.
// FailTasks run in the order added, and the list should be short.
func (r *Robot) FailTask(name string, cmdargs ...string) RetVal {
	c := r.getContext()
	if c.stage != primaryTasks {
		return InvalidStage
	}
	t := c.tasks.getTaskByName(name)
	if t == nil {
		return TaskNotFound
	}
	_, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	var command string
	var args []string
	if isPlugin {
		if len(cmdargs) == 0 {
			return MissingArguments
		}
		if len(cmdargs[0]) == 0 {
			return MissingArguments
		}
		command, args = cmdargs[0], cmdargs[1:]
	} else {
		command = "run"
		args = cmdargs
	}
	ts := taskSpec{
		Name:      name,
		Command:   command,
		Arguments: args,
		task:      t,
	}
	c.failTasks = append(c.failTasks, ts)
	return Ok
}

// GetParameter retrieves the value of a parameter for a namespace. Only useful
// for Go plugins; external scripts have all parameters for the NameSpace stored
// as environment variables. Note that runtasks.go populates the environment
// with Stored parameters, too. So GetParameter is useful for both short-term
// parameters in a pipeline, and for getting long-term parameters such as
// credentials.
func (r *Robot) GetParameter(key string) string {
	c := r.getContext()
	value, ok := c.taskenvironment[key]
	if ok {
		return value
	}
	return ""
}

// Elevate lets a plugin request elevation on the fly. When immediate = true,
// the elevator should always prompt for 2fa; otherwise a configured timeout
// should apply.
func (r *Robot) Elevate(immediate bool) bool {
	c := r.getContext()
	task, _, _ := getTask(c.currentTask)
	retval := c.elevate(task, immediate)
	if retval == Success {
		return true
	}
	return false
}

// Fixed is a deprecated convenience function for sending a message with fixed width
// font.
func (r *Robot) Fixed() *Robot {
	nr := *r
	nr.Format = Fixed
	return &nr
}

// MessageFormat returns a robot object with the given format, most likely for a
// plugin that will mostly use e.g. Variable format.
func (r *Robot) MessageFormat(f MessageFormat) *Robot {
	nr := *r
	nr.Format = f
	return &nr
}

// Direct is a convenience function for initiating a DM conversation with a
// user. Created initially so a plugin could prompt for a password in a DM.
func (r *Robot) Direct() *Robot {
	nr := *r
	nr.Channel = ""
	return &nr
}

// Pause is a convenience function to pause some fractional number of seconds.
func (r *Robot) Pause(s float64) {
	ms := time.Duration(s * float64(1000))
	time.Sleep(ms * time.Millisecond)
}

// RandomString is a convenience function for returning a random string
// from a slice of strings, so that replies can vary.
func (r *Robot) RandomString(s []string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	return s[random.Intn(l)]
}

// RandomInt uses the robot's seeded random to return a random int 0 <= retval < n
func (r *Robot) RandomInt(n int) int {
	return random.Intn(n)
}

// GetBotAttribute returns an attribute of the robot or "" if unknown.
// Current attributes:
// name, alias, fullName, contact
func (r *Robot) GetBotAttribute(a string) *AttrRet {
	a = strings.ToLower(a)
	botCfg.RLock()
	defer botCfg.RUnlock()
	ret := Ok
	var attr string
	switch a {
	case "name":
		attr = botCfg.name
	case "fullname", "realname":
		attr = botCfg.fullName
	case "alias":
		attr = string(botCfg.alias)
	case "email":
		attr = botCfg.email
	case "contact", "admin", "admincontact":
		attr = botCfg.adminContact
	case "protocol":
		attr = r.Protocol.String()
	default:
		ret = AttributeNotFound
	}
	return &AttrRet{attr, ret}
}

// GetUserAttribute returns a AttrRet with
// - The string Attribute of a user, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: supplement data with gopherbot.yaml user's table, if an
// admin wants to supplment whats available from the protocol.
func (r *Robot) GetUserAttribute(u, a string) *AttrRet {
	a = strings.ToLower(a)
	attr, ret := botCfg.GetProtocolUserAttribute(u, a)
	return &AttrRet{attr, ret}
}

// messageHeard sends a typing notification
func (r *Robot) messageHeard() {
	botCfg.MessageHeard(r.User, r.Channel)
}

// GetSenderAttribute returns a AttrRet with
// - The string Attribute of the sender, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: (see above)
func (r *Robot) GetSenderAttribute(a string) *AttrRet {
	a = strings.ToLower(a)
	switch a {
	case "name", "username", "handle", "user", "user name":
		return &AttrRet{r.User, Ok}
	default:
		attr, ret := botCfg.GetProtocolUserAttribute(r.User, a)
		return &AttrRet{attr, ret}
	}
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
func (r *Robot) GetTaskConfig(dptr interface{}) RetVal {
	c := r.getContext()
	task, _, _ := getTask(c.currentTask)
	if task.config == nil {
		Log(Debug, fmt.Sprintf("Task \"%s\" called GetTaskConfig, but no config was found.", task.name))
		return NoConfigFound
	}
	tp := reflect.ValueOf(dptr)
	if tp.Kind() != reflect.Ptr {
		Log(Debug, fmt.Sprintf("Task \"%s\" called GetTaskConfig, but didn't pass a double-pointer to a struct", task.name))
		return InvalidDblPtr
	}
	p := reflect.Indirect(tp)
	if p.Kind() != reflect.Ptr {
		Log(Debug, fmt.Sprintf("Task \"%s\" called GetTaskConfig, but didn't pass a double-pointer to a struct", task.name))
		return InvalidDblPtr
	}
	if p.Type() != reflect.ValueOf(task.config).Type() {
		Log(Debug, fmt.Sprintf("Task \"%s\" called GetTaskConfig with an invalid double-pointer", task.name))
		return InvalidCfgStruct
	}
	p.Set(reflect.ValueOf(task.config))
	return Ok
}

// Log logs a message to the robot's log file (or stderr) if the level
// is lower than or equal to the robot's current log level
func (r *Robot) Log(l LogLevel, v ...interface{}) {
	c := r.getContext()
	if c.logger != nil {
		c.logger.Log("LOG:" + logLevelToStr(l) + " " + fmt.Sprintln(v...))
	}
	Log(l, v...)
}

// SendChannelMessage lets a plugin easily send a message to an arbitrary
// channel. Use Robot.Fixed().SendChannelMessage(...) for fixed-width
// font.
func (r *Robot) SendChannelMessage(channel, msg string) RetVal {
	return botCfg.SendProtocolChannelMessage(channel, msg, r.Format)
}

// SendUserChannelMessage lets a plugin easily send a message directed to
// a specific user in a specific channel without fiddling with the robot
// object. Use Robot.Fixed().SencChannelMessage(...) for fixed-width
// font.
func (r *Robot) SendUserChannelMessage(user, channel, msg string) RetVal {
	return botCfg.SendProtocolUserChannelMessage(user, channel, msg, r.Format)
}

// SendUserMessage lets a plugin easily send a DM to a user. If a DM
// isn't possible, the connector should message the user in a channel.
func (r *Robot) SendUserMessage(user, msg string) RetVal {
	return botCfg.SendProtocolUserMessage(user, msg, r.Format)
}

// Reply directs a message to the user
func (r *Robot) Reply(msg string) RetVal {
	if r.Channel == "" {
		return botCfg.SendProtocolUserMessage(r.User, msg, r.Format)
	}
	return botCfg.SendProtocolUserChannelMessage(r.User, r.Channel, msg, r.Format)
}

// Say just sends a message to the user or channel
func (r *Robot) Say(msg string) RetVal {
	if r.Channel == "" {
		return botCfg.SendProtocolUserMessage(r.User, msg, r.Format)
	}
	return botCfg.SendProtocolChannelMessage(r.Channel, msg, r.Format)
}
