package bot

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

/* botcontext.go - internal methods on botContexts */

// Global robot run number (incrementing int)
var botRunID = struct {
	idx int
	sync.Mutex
}{
	0,
	sync.Mutex{},
}

// Global persistent maps of Robots running, for Robot lookups in http.go
var activeRobots = struct {
	i map[int]*botContext
	sync.RWMutex
}{
	make(map[int]*botContext),
	sync.RWMutex{},
}

// getBotContextStr is used to look up a botContext in httpd.go, so we do the
// string conversion here. Note that 0 is never a valid bot id, and this will
// return nil for any failures.
func getBotContextStr(id string) *botContext {
	idx, _ := strconv.Atoi(id)
	activeRobots.RLock()
	bot, _ := activeRobots.i[idx]
	activeRobots.RUnlock()
	return bot
}

// getBotContextInt is used to look up a botContext from a Robot in when needed.
// Note that 0 is never a valid bot id, and this will return nil in that case.
func getBotContextInt(idx int) *botContext {
	activeRobots.RLock()
	bot, _ := activeRobots.i[idx]
	activeRobots.RUnlock()
	return bot
}

// Assign a bot run number and register it in the global hash of running
// robots. Should be called before running plugins.
func (c *botContext) registerActive(parent *botContext) {
	if c.Incoming != nil {
		c.Protocol, _ = getProtocol(c.Incoming.Protocol)
	}
	c.Format = currentCfg.defaultMessageFormat
	c.environment["GOPHER_HTTP_POST"] = "http://" + listenPort

	// Only needed for bots not created by IncomingMessage
	if c.maps == nil {
		currentUCMaps.Lock()
		c.maps = currentUCMaps.ucmap
		currentUCMaps.Unlock()
	}
	if len(c.ProtocolUser) == 0 && len(c.User) > 0 {
		if idRegex.MatchString(c.User) {
			c.ProtocolUser = c.User
		} else if ui, ok := c.maps.user[c.User]; ok {
			c.ProtocolUser = bracket(ui.UserID)
			c.BotUser = ui.BotUser
		} else {
			c.ProtocolUser = c.User
		}
	}
	if len(c.ProtocolChannel) == 0 && len(c.Channel) > 0 {
		if idRegex.MatchString(c.Channel) {
			c.ProtocolChannel = c.Channel
		} else if ci, ok := c.maps.channel[c.Channel]; ok {
			c.ProtocolChannel = bracket(ci.ChannelID)
		} else {
			c.ProtocolChannel = c.Channel
		}
	}

	c.nextTasks = make([]TaskSpec, 0)
	c.finalTasks = make([]TaskSpec, 0)

	c.environment["GOPHER_INSTALLDIR"] = installPath

	botRunID.Lock()
	botRunID.idx++
	if botRunID.idx == 0 {
		botRunID.idx = 1
	}
	c.id = botRunID.idx
	c.environment["GOPHER_CALLER_ID"] = fmt.Sprintf("%d", c.id)
	botRunID.Unlock()

	activeRobots.Lock()
	if parent != nil {
		parent.child = c
		c.parent = parent
	}
	activeRobots.i[c.id] = c
	activeRobots.Unlock()
	c.active = true
}

// deregister must be called for all registered Robots to prevent a memory leak.
func (c *botContext) deregister() {
	activeRobots.Lock()
	delete(activeRobots.i, c.id)
	activeRobots.Unlock()
	c.active = false
}

// makeRobot returns a *Robot for plugins; the id lets Robot methods
// get a reference back to the original context.
func (c *botContext) makeRobot() Robot {
	return Robot{
		&robot.Message{
			User:            c.User,
			ProtocolUser:    c.ProtocolUser,
			Channel:         c.Channel,
			ProtocolChannel: c.ProtocolChannel,
			Format:          c.Format,
			Protocol:        c.Protocol,
			Incoming:        c.Incoming,
		},
		c.id,
	}
}

// clone() is a convenience function to clone the current context before
// starting a new goroutine for startPipeline. Used by e.g. triggered jobs,
// SpawnJob(), and runPipeline for sub-jobs.
func (c *botContext) clone() *botContext {
	c.RLock()
	clone := &botContext{
		User:             c.User,
		ProtocolUser:     c.ProtocolUser,
		Channel:          c.Channel,
		ProtocolChannel:  c.ProtocolChannel,
		Incoming:         c.Incoming,
		directMsg:        c.directMsg,
		BotUser:          c.BotUser,
		listedUser:       c.listedUser,
		_pipeName:        c._pipeName,
		_pipeDesc:        c._pipeDesc,
		ptype:            c.ptype,
		cfg:              c.cfg,
		tasks:            c.tasks,
		maps:             c.maps,
		repositories:     c.repositories,
		automaticTask:    c.automaticTask,
		_elevated:        c._elevated,
		Protocol:         c.Protocol,
		Format:           c.Format,
		msg:              c.msg,
		workingDirectory: "",
		_environment:     make(map[string]string),
	}
	c.RUnlock()
	return clone
}

// botContext is created for each incoming message, in a separate goroutine that
// persists for the life of the message, until finally a plugin runs
// (or doesn't). It could also be called Context, or PipelineState; but for
// use by plugins, it's best left as Robot.
type botContext struct {
	User             string                      // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel          string                      // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	ProtocolUser     string                      // The username or <userid> to be sent in connector methods
	ProtocolChannel  string                      // the channel name or <channelid> where the message originated
	Protocol         robot.Protocol              // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	Incoming         *robot.ConnectorMessage     // raw struct of message sent by connector; interpret based on protocol. For Slack this is a *slack.MessageEvent
	Format           robot.MessageFormat         // robot's default message format
	workingDirectory string                      // directory where tasks run relative to $(pwd)
	baseDirectory    string                      // base for this pipeline relative to $(pwd), depends on `Homed`, affects SetWorkingDirectory
	privileged       bool                        // privileged jobs flip this flag, causing tasks in the pipeline to run in cfgdir
	id               int                         // incrementing index of Robot threads
	tasks            *taskList                   // Pointers to current task configuration at start of pipeline
	maps             *userChanMaps               // Pointer to current user / channel maps struct
	repositories     map[string]robot.Repository // Set of configured repositories
	cfg              *configuration              // Active configuration when this context was created
	BotUser          bool                        // set for bots/programs that should never match ambient messages
	listedUser       bool                        // set for users listed in the UserRoster; ambient messages don't match unlisted users by default
	isCommand        bool                        // Was the message directed at the robot, dm or by mention
	directMsg        bool                        // if the message was sent by DM
	msg              string                      // the message text sent
	automaticTask    bool                        // set for scheduled & triggers jobs, where user security restrictions don't apply
	history          robot.HistoryProvider       // history provider for generating the logger
	timeZone         *time.Location              // for history timestamping
	logger           robot.HistoryLogger         // where to send stdout / stderr
	active           bool                        // whether this context has been registered as active
	ptype            pipelineType                // what started this pipeline

	// Parent and child values protected by the activeRobots lock
	update            chan interface{}  // Channel for serializing ... ?
	__parent, __child *botContext       // for sub-job contexts
	_elevated         bool              // set when required elevation succeeds
	_environment      map[string]string // environment vars set for each job/plugin in the pipeline
	_taskenvironment  map[string]string // per-task environment for Go plugins
	_stage            pipeStage         // which pipeline is being run; primaryP, finalP, failP
	_jobInitialized   bool              // whether a job has started
	_jobName          string            // name of the running job
	_jobChannel       string            // channel where job updates are posted
	_nsExtension      string            // extended namespace
	_runIndex         int               // run number of a job
	_verbose          bool              // flag if initializing job was verbose
	_nextTasks        []TaskSpec        // tasks in the pipeline
	_finalTasks       []TaskSpec        // clean-up tasks that always run when the pipeline ends
	_failTasks        []TaskSpec        // clean-up tasks that run when a pipeline fails

	_failedTask, failedTaskDescription string // set when a task fails

	_pipeName, _pipeDesc string      // name and description of task that started pipeline
	_currentTask         interface{} // pointer to currently executing task
	_taskName            string      // name of current task
	_taskDesc            string      // description for same
	_osCmd               *exec.Cmd   // running Command, for aborting a pipeline

	_exclusiveTag  string // tasks with the same exclusiveTag never run at the same time
	_exclusive     bool   // indicates task was running exclusively
	_queueTask     bool   // whether to queue up if Exclusive call failed
	_abortPipeline bool   // Exclusive request failed w/o queueTask
}
