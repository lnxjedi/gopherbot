package bot

import (
	"crypto/rand"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/lnxjedi/robot"
)

/* pipeContext.go - internal methods on pipeContexts

Note that the pipeContext includes a sync.mutex, since it keeps pipeline state
and changes over the course of a running pipeline. Since the pipeContext is
nested in the worker, it can be locked with w.Lock() and modified with simple
assignments to w.xxx.
*/

// Global context run number (incrementing int)
var workerID = struct {
	idx int
	sync.Mutex
}{
	0,
	sync.Mutex{},
}

// Get the next context ID
func getWorkerID() int {
	workerID.Lock()
	workerID.idx++
	if workerID.idx == maxIndex {
		workerID.idx = 1
	}
	ctxid := workerID.idx
	workerID.Unlock()
	return ctxid
}

// Global persistent maps of pipelines running, only used for listing/forcibly
// stopping pipelines.
var activePipelines = struct {
	i    map[int]*worker
	eids map[string]struct{}
	sync.Mutex
}{
	make(map[int]*worker),
	make(map[string]struct{}),
	sync.Mutex{},
}

// Assign a worker an external ID and register it in the global map of running
// pipelines.
func (w *worker) registerActive(parent *worker) {
	// Only needed for bots not created by IncomingMessage
	if w.maps == nil {
		currentUCMaps.Lock()
		w.maps = currentUCMaps.ucmap
		currentUCMaps.Unlock()
	}
	if len(w.ProtocolUser) == 0 && len(w.User) > 0 {
		if idRegex.MatchString(w.User) {
			w.ProtocolUser = w.User
		} else if ui, ok := w.maps.user[w.User]; ok {
			w.ProtocolUser = bracket(ui.UserID)
			w.BotUser = ui.BotUser
		} else {
			w.ProtocolUser = w.User
		}
	}
	if len(w.ProtocolChannel) == 0 && len(w.Channel) > 0 {
		if idRegex.MatchString(w.Channel) {
			w.ProtocolChannel = w.Channel
		} else if ci, ok := w.maps.channel[w.Channel]; ok {
			w.ProtocolChannel = bracket(ci.ChannelID)
		} else {
			w.ProtocolChannel = w.Channel
		}
	}

	activePipelines.Lock()
	if len(w.eid) == 0 {
		var eid string
		for {
			// 4 bytes of entropy per pipeline
			b := make([]byte, 4)
			rand.Read(b)
			eid = fmt.Sprintf("%02x%02x%02x%02x", b[0], b[1], b[2], b[3])
			if _, ok := activePipelines.eids[eid]; !ok {
				activePipelines.eids[eid] = struct{}{}
				break
			}
		}
		w.eid = eid
	}
	if parent != nil {
		parent._child = w
		w._parent = parent
	}
	activePipelines.i[w.id] = w
	activePipelines.Unlock()
	w.active = true
}

// deregister must be called for all registered Robots to prevent a memory leak.
func (w *worker) deregister() {
	w.Lock()
	id := w.id
	eid := w.eid
	w.active = false
	w.Unlock()
	activePipelines.Lock()
	delete(activePipelines.i, id)
	delete(activePipelines.eids, eid)
	activePipelines.Unlock()
}

// pipeContext is created for each running pipeline, and holds the state
// for the pipeline.
type pipeContext struct {
	// Parent and child values protected by the activePipelines lock
	_parent, _child  *worker
	workingDirectory string            // directory where tasks run relative to $(pwd)
	baseDirectory    string            // base for this pipeline relative to $(pwd), depends on `Homed`, affects SetWorkingDirectory
	eid              string            // unique ID for external tasks
	active           bool              // whether this context has been registered as active
	environment      map[string]string // environment vars set for each job/plugin in the pipeline
	runIndex         int               // run number of a job
	histName         string            // GetLog(histName, index) can be used in final/fail pipes
	verbose          bool              // flag if initializing job was verbose
	nextTasks        []TaskSpec        // tasks in the pipeline
	finalTasks       []TaskSpec        // clean-up tasks that always run when the pipeline ends
	failTasks        []TaskSpec        // clean-up tasks that run when a pipeline fails
	finalFailed      []string          // list of task names of final tasks that failed
	taskName         string            // name of current task
	taskDesc         string            // description for same
	taskType         string            // one of task, plugin, job
	plugCommand      string            // plugin command if type=plugin, else blank
	taskArgs         []string          // args for current task
	osCmd            *exec.Cmd         // running Command, for aborting a pipeline
	exclusiveTag     string            // tasks with the same exclusiveTag never run at the same time
	queueTask        bool              // whether to queue up if Exclusive call failed
	abortPipeline    bool              // Exclusive request failed w/o queueTask
	// Stuff we want to copy in makeRobot
	privileged         bool                // privileged jobs flip this flag, causing tasks in the pipeline to run in cfgdir
	timeZone           *time.Location      // for history timestamping
	logger             robot.HistoryLogger // where to send stdout / stderr
	ptype              pipelineType        // what started this pipeline
	elevated           bool                // set when required elevation succeeds
	stage              pipeStage           // which pipeline is being run; primaryP, finalP, failP
	jobInitialized     bool                // whether a job has started
	jobName            string              // name of the running job
	nameSpace          string              // namespace for the pipeline, used by exclusive
	pipeName, pipeDesc string              // name and description of task that started pipeline
	nsExtension        string              // extended namespace
	currentTask        interface{}         // pointer to currently executing task
	exclusive          bool                // indicates task was running exclusively
}

func (c *pipeContext) section(name, info string) {
	if c.logger != nil {
		c.logger.Line("*** " + name + " - " + info)
	}
}
