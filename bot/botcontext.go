package bot

import (
	"fmt"
	"rand"
	"strconv"
)

/* robot.go - internal methods on the Robot object */

// Global robot run number (incrementing int)
var botRunID = struct {
	idx int
	sync.Mutex
}{
	rand.Int(),
	sync.Mutex{},
}

// Global persistent maps of Robots running, for Robot lookups in http.go
var activeRobots = struct {
	i map[int]*Robot
	sync.Mutex
}{
	make(map[int]*Robot),
	sync.Mutex{},
}

// getRobot is used to look up a robot in httpd.go, so we do the string
// conversion here. Note that 0 is never a valid bot id, and this will return
// nil for any failures.
func getRobot(id string) *botContext {
	idx, _ := strconv.Atoi(id)
	activeRobots.RLock()
	bot, _ := activeRobots.i[idx]
	activeRobots.RUnlock()
}

// Assign a bot run number and register it in the global hash of running
// robots. Should be called before running plugins
func (bot *botContext) registerActive() {
	botRunID.Lock()
	botRunID.idx++
	if botRunID.idx == 0 {
		botRunID.idx = 1
	}
	bot.id = botRunID.idx
	botRunID.Unlock()
	activeRobots.Lock()
	activeRobots.i[bot.id] = bot
	activeRobots.Unlock()
}

// deregister must be called for all registered Robots to prevent a memory leak.
func (bot *botContext) deregister() {
	activeRobots.Lock()
	delete(activeRobots.i, bot.id)
	activeRobots.Unlock()
}

// botContext is created for each incoming message, in a separate goroutine that
// persists for the life of the message, until finally a plugin runs
// (or doesn't). It could also be called Context, or PipelineState; but for
// use by plugins, it's best left as Robot.
type botContext struct {
	User           string            // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel        string            // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Protocol       Protocol          // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	RawMsg         interface{}       // raw struct of message sent by connector; interpret based on protocol. For Slack this is a *slack.MessageEvent
	id             int               // incrementing index of Robot threads
	tasks          taskList          // Pointers to current task configuration at start of pipeline
	currentTask    interface{}       // pointer to currently executing task
	isCommand      bool              // Was the message directed at the robot, dm or by mention
	directMsg      bool              // if the message was sent by DM
	msg            string            // the message text sent
	bypassSecurity bool              // set for scheduled jobs, where user security restrictions don't apply
	elevated       bool              // set when required elevation succeeds
	environment    map[string]string // environment vars set for each job/plugin in the pipeline
}
