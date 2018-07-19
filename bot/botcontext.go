package bot

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

/* robot.go - internal methods on the Robot object */

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
// robots. Should be called before running plugins
func (c *botContext) registerActive() {
	robot.RLock()
	c.Protocol = setProtocol(robot.protocol)
	c.Format = robot.defaultMessageFormat
	c.environment["GOPHER_HTTP_POST"] = "http://" + robot.port
	robot.RUnlock()
	c.nextTasks = make([]taskSpec, 0)
	c.finalTasks = make([]taskSpec, 0)
	botRunID.Lock()
	botRunID.idx++
	if botRunID.idx == 0 {
		botRunID.idx = 1
	}
	c.id = botRunID.idx
	c.environment["GOPHER_INSTALLDIR"] = installPath
	if len(configPath) > 0 {
		c.environment["GOPHER_CONFIGDIR"] = configPath
	} else {
		c.environment["GOPHER_CONFIGDIR"] = installPath
	}
	c.environment["GOPHER_CALLER_ID"] = fmt.Sprintf("%d", c.id)
	botRunID.Unlock()
	activeRobots.Lock()
	activeRobots.i[c.id] = c
	activeRobots.Unlock()
}

// deregister must be called for all registered Robots to prevent a memory leak.
func (c *botContext) deregister() {
	activeRobots.Lock()
	delete(activeRobots.i, c.id)
	activeRobots.Unlock()
}

// makeRobot returns
func (c *botContext) makeRobot() *Robot {
	return &Robot{
		User:     c.User,
		Channel:  c.Channel,
		Format:   c.Format,
		Protocol: c.Protocol,
		RawMsg:   c.RawMsg,
		id:       c.id,
	}
}

// botContext is created for each incoming message, in a separate goroutine that
// persists for the life of the message, until finally a plugin runs
// (or doesn't). It could also be called Context, or PipelineState; but for
// use by plugins, it's best left as Robot.
type botContext struct {
	User             string            // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel          string            // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Protocol         Protocol          // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	RawMsg           interface{}       // raw struct of message sent by connector; interpret based on protocol. For Slack this is a *slack.MessageEvent
	Format           MessageFormat     // robot's default message format
	nameSpace        string            // memory namespace for this pipeline
	workingDirectory string            // directory where tasks run
	id               int               // incrementing index of Robot threads
	tasks            taskList          // Pointers to current task configuration at start of pipeline
	isCommand        bool              // Was the message directed at the robot, dm or by mention
	directMsg        bool              // if the message was sent by DM
	msg              string            // the message text sent
	automaticTask    bool              // set for scheduled & triggers jobs, where user security restrictions don't apply
	elevated         bool              // set when required elevation succeeds
	environment      map[string]string // environment vars set for each job/plugin in the pipeline
	taskenvironment  map[string]string // per-task environment for Go plugins

	stage          pipeStage  // which pipeline is being run; primaryP, finalP, failP
	jobInitialized bool       // whether a job has started
	runIndex       int        // run number of a job
	jobChannel     string     // channel for reporting job status
	verbose        bool       // flag if initializing job was verbose
	nextTasks      []taskSpec // tasks in the pipeline
	finalTasks     []taskSpec // clean-up tasks that always run when the pipeline ends
	failTasks      []taskSpec // clean-up tasks that run when a pipeline fails

	history  HistoryProvider // history provider for generating the logger
	timeZone *time.Location  // for history timestamping
	logger   HistoryLogger   // where to send stdout / stderr

	sync.Mutex                     // Protects access to the items below
	pipeName, pipeDesc string      // name and description of task that started pipeline
	currentTask        interface{} // pointer to currently executing task
	taskName           string      // name of current task
	taskDesc           string      // description for same
	osCmd              *exec.Cmd   // running Command, for aborting a pipeline

	exclusiveTag  string // tasks with the same exclusiveTag never run at the same time
	exclusive     bool   // indicates task was running exclusively
	queueTask     bool   // whether to queue up if Exclusive call failed
	abortPipeline bool   // Exclusive request failed w/o queueTask
}
