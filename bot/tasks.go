package bot

import (
	"encoding/json"
	"log"
	"regexp"
	"runtime"
	"sync"

	"github.com/lnxjedi/robot"
)

// Regex for task/job/plugin/NameSpace names. NOTE: if this changes,
// command regexes in jobbuiltins.go will need to be changed.
// Identifiers can be letters, numbers & underscores only, mainly so
// brain functions can use ':' as a separator.
const identifierRegex = `[A-Za-z][\w-]*`

var identifierRe = regexp.MustCompile(identifierRegex)

// Global persistent map of task/namespace name to unique ID
// TODO: rename this when errors are cleared
var taskNameIDmap = struct {
	m map[string]string
	sync.Mutex
}{
	make(map[string]string),
	sync.Mutex{},
}

type taskList struct {
	t []interface{}
	// nameMap - map of every task, job, plugin and namespace to t[] index;
	// namespaces are all idx==0
	nameMap map[string]int
	idMap   map[string]int // task ID to task number
	// map of namespace name to NameSpace, updated on every load
	nameSpaces map[string]NameSpace
}

func getTask(t interface{}) (*Task, *Plugin, *Job) {
	p, ok := t.(*Plugin)
	if ok {
		return p.Task, p, nil
	}
	j, ok := t.(*Job)
	if ok {
		return j.Task, nil, j
	}
	return t.(*Task), nil, nil
}

func (tl *taskList) getTaskByName(name string) interface{} {
	if len(name) == 0 {
		_, file, line, _ := runtime.Caller(1)
		Log(robot.Error, "Invalid 0-length task from caller: %s, line %d", file, line)
	}
	ti, ok := tl.nameMap[name]
	if !ok {
		Log(robot.Error, "task '%s' not found calling getTaskByName", name)
		return nil
	}
	if ti == 0 {
		Log(robot.Error, "'%s' refers to a namespace in getTaskByName", name)
		return nil
	}
	task := tl.t[ti]
	return task
}

// true if name refers to a NameSpace, false if not or doesn't exist
func (tl *taskList) isNamespace(name string) (ok bool) {
	_, ok = tl.nameSpaces[name]
	return
}

// TaskSpec is the structure for ScheduledJobs (robot.yaml) and AddTask (robot method)
type TaskSpec struct {
	Name      string // name of the job or plugin
	Command   string // plugins only
	Arguments []string
	task      interface{} // populated in AddTask
}

// TaskSettings struct used for configuration of: ExternalPlugins, ExternalJobs,
// ExternalTasks, GoPlugins, GoJobs, GoTasks and NameSpaces in robot.yaml.
// Not every field is used in every case.
type TaskSettings struct {
	Name, Path, Description, NameSpace string
	Disabled                           bool
	Homed                              bool
	Privileged                         *bool
	Parameters                         []robot.Parameter
}

// LoadableModule struct for loading external modules.
type LoadableModule struct {
	Name, Path, Description string
	Disabled                bool
}

// ScheduledTask items defined in robot.yaml, mostly for scheduled jobs
type ScheduledTask struct {
	Schedule string // timespec for https://godoc.org/github.com/robfig/cron
	TaskSpec
}

// PluginHelp specifies keywords and help text for the 'bot help system
type PluginHelp struct {
	Keywords []string // match words for 'help XXX'
	Helptext []string // help string to give for the keywords, conventionally starting with (bot) for commands or (hear) when the bot needn't be addressed directly
}

// InputMatcher specifies the command or message to match for a plugin
type InputMatcher struct {
	Regex    string         // The regular expression string to match - bot adds ^\w* & \w*$
	Command  string         // The name of the command to pass to the plugin with it's arguments
	Label    string         // ReplyMatchers use "Label" instead of "Command"
	Contexts []string       // label the contexts corresponding to capture groups, for supporting "it" & optional args
	re       *regexp.Regexp // The compiled regular expression. If the regex doesn't compile, the 'bot will log an error
}

// JobTrigger specifies a user and message to trigger a job
type JobTrigger struct {
	Regex   string         // The regular expression string to match - bot adds ^\w* & \w*$
	User    string         // required user to trigger this job, normally git-activated webhook or integration
	Channel string         // required channel for the trigger
	re      *regexp.Regexp // The compiled regular expression. If the regex doesn't compile, the 'bot will log an error
}

// NameSpace just stores a name, description, and parameters - they cannot be run.
type NameSpace struct {
	name        string            // name of the shared namespace
	Description string            // optional description of the shared namespace
	Parameters  []robot.Parameter // Parameters for the shared namespace
}

// Task configuration is common to tasks, plugins or jobs. Any task, plugin or job can call bot methods. Note that tasks are only defined
// in robot.yaml, and no external configuration is read in.
type Task struct {
	name         string            // name of job or plugin; unique by type, but job & plugin can share
	taskType     taskType          // taskGo or taskExternal
	Path         string            // Path to the external executable for jobs or Plugtype=taskExternal only
	NameSpace    string            // callers that share namespace share long-term memories and environment vars; defaults to name if not otherwise set
	Parameters   []robot.Parameter // Fixed parameters for a given job; many jobs will use the same script with differing parameters
	Description  string            // description of job or plugin
	AllowDirect  bool              // Set this true if this plugin can be accessed via direct message
	DirectOnly   bool              // Set this true if this plugin ONLY accepts direct messages
	Channel      string            // channel where a job can be interracted with, channel where a scheduled task (job or plugin) runs
	Channels     []string          // plugins only; Channels where the plugin is available - rifraf like "memes" should probably only be in random, but it's configurable. If empty uses DefaultChannels
	AllChannels  bool              // If the Channels list is empty and AllChannels is true, the plugin should be active in all the channels the bot is in
	RequireAdmin bool              // Set to only allow administrators to access a plugin / run job
	Users        []string          // If non-empty, list of all the users with access to this plugin
	Elevator     string            // Use an elevator other than the DefaultElevator
	Authorizer   string            // a plugin to call for authorizing users, should handle groups, etc.
	AuthRequire  string            // an optional group/role name to be passed to the Authorizer plugin, for group/role-based authorization determination
	// taskID        string            // 32-char random ID for identifying plugins/jobs
	ReplyMatchers []InputMatcher  // store this here for prompt*reply methods
	Config        json.RawMessage // Arbitrary Plugin configuration, will be stored and provided in a thread-safe manner via GetTaskConfig()
	config        interface{}     // A pointer to an empty struct that the bot can Unmarshal custom configuration into
	Disabled      bool
	reason        string // why this job/plugin is disabled
	// Privileged jobs/plugins run with the privileged UID, privileged tasks
	// require privileged pipelines.
	Privileged bool
	// Homed for jobs/plugins starts the pipeline with c.basePath = ".", Homed tasks
	// always run in ".", e.g. "ssh-init"
	Homed bool
}

// Job - configuration only applicable to jobs. Read in from conf/jobs/<job>.yaml, which can also include anything from a Task.
type Job struct {
	Quiet       bool           // whether to quash "job started/ended" messages
	HistoryLogs int            // how many runs of this job/plugin to keep history for
	Triggers    []JobTrigger   // user/regex that triggers a job, e.g. a git-activated webhook or integration
	Arguments   []InputMatcher // list of arguments to prompt the user for
	*Task
}

// Plugin specifies the structure of a plugin configuration - plugins should include an example / default config. Custom plugin configuration
// will be loaded from conf/plugins/<plugin>.yaml, which can also include anything from a Task.
type Plugin struct {
	AdminCommands            []string       // A list of commands only a bot admin can use
	ElevatedCommands         []string       // Commands that require elevation, usually via 2fa
	ElevateImmediateCommands []string       // Commands that always require elevation promting, regardless of timeouts
	AuthorizedCommands       []string       // Which commands to authorize
	AuthorizeAllCommands     bool           // when ALL commands need to be authorized
	Help                     []PluginHelp   // All the keyword sets / help texts for this plugin
	CommandMatchers          []InputMatcher // Input matchers for messages that need to be directed to the 'bot
	MessageMatchers          []InputMatcher // Input matchers for messages the 'bot hears even when it's not being spoken to
	CatchAll                 bool           // Whenever the robot is spoken to, but no plugin matches, plugins with CatchAll=true get called with command="catchall" and argument=<full text of message to robot>
	MatchUnlisted            bool           // Set to true if ambient messages matches should be checked for users not listed in the UserRoster
	*Task
}

var pluginHandlers = make(map[string]robot.PluginHandler)
var jobHandlers = make(map[string]robot.JobHandler)
var taskHandlers = make(map[string]robot.TaskHandler)

// stopRegistrations is set "true" when the bot is created to prevent registration outside of init functions
var stopRegistrations = false

// initialize sends the "init" command to every plugin
func initializePlugins() {
	currentCfg.RLock()
	cfg := currentCfg.configuration
	tasks := currentCfg.taskList
	protocol := currentCfg.protocol
	currentCfg.RUnlock()
	state.Lock()
	if !state.shuttingDown {
		state.Unlock()
		for _, t := range tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			if task.Disabled {
				continue
			}
			w := &worker{
				cfg:           cfg,
				tasks:         tasks,
				Protocol:      getProtocol(protocol),
				automaticTask: true,
				id:            getWorkerID(),
			}
			Log(robot.Info, "Initializing plugin: %s", task.name)
			go w.startPipeline(nil, t, plugCommand, "init")
		}
	} else {
		state.Unlock()
	}
}

// registerTask centralizes the sanity checking logic for RegisterPlugin,
// RegisterJob and RegisterTask
func registerTask(name string) *Task {
	if stopRegistrations {
		return nil
	}
	if !identifierRe.MatchString(name) {
		log.Fatalf("Name '%s' doesn't match name regex '%s'", name, identifierRe.String())
	}
	if name == "bot" {
		log.Fatalf("Illegal name registration for 'bot'")
	}
	if _, ok := currentCfg.nameMap[name]; ok {
		log.Fatalf("Go task '%s' name collision with other task/job/plugin/namespace", name)
	}
	task := &Task{
		name:     name,
		taskType: taskGo,
	}
	return task
}

// addTask adds the registered task to the global list
func (tl *taskList) addTask(t interface{}) {
	task, _, _ := getTask(t)
	idx := len(tl.t)
	tl.t = append(tl.t, t)
	tl.nameMap[task.name] = idx
	tl.idMap[task.name] = idx
}

// RegisterPlugin allows Go plugins to register a PluginHandler in a func init().
// Also called for new plugins loaded with a loadable module.
// When the bot initializes, it will call each plugin's handler with a command
// "init", empty channel, the bot's username, and no arguments, so the plugin
// can store this information for, e.g., scheduled jobs.
// See robot/structs.go for the pluginHandlers definition.
func RegisterPlugin(name string, plug robot.PluginHandler) {
	task := registerTask(name)
	if task == nil {
		return
	}
	plugin := &Plugin{
		Task: task,
	}
	currentCfg.addTask(plugin)
	pluginHandlers[name] = plug
}

// RegisterJob registers a Go job
func RegisterJob(name string, gojob robot.JobHandler) {
	task := registerTask(name)
	if task == nil {
		return
	}
	job := &Job{
		Task: task,
	}
	currentCfg.addTask(job)
	jobHandlers[name] = gojob
}

// RegisterTask registers a Go task. If prevRequired is set, the task can
// only be added to a privileged pipeline.
func RegisterTask(name string, privRequired bool, gotask robot.TaskHandler) {
	task := registerTask(name)
	if task == nil {
		return
	}
	task.Privileged = privRequired
	currentCfg.addTask(task)
	taskHandlers[name] = gotask
}
