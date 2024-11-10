package bot

import (
	"regexp"
	"runtime"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
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
	nameSpaces map[string]ParameterSet
	// map of parameterset name to NameSpace (re-using identical data structure)
	parameterSets map[string]ParameterSet
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
		Log(robot.Error, "Task '%s' not found calling getTaskByName", name)
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
	Name      string      `yaml:"Name"`      // Name of the job or plugin
	Command   string      `yaml:"Command"`   // Plugins only
	Arguments []string    `yaml:"Arguments"` // Arguments for the task
	task      interface{} `yaml:"-"`         // Populated in AddTask
}

// TaskSettings struct used for configuration of: ExternalPlugins, ExternalJobs,
// ExternalTasks, GoPlugins, GoJobs, GoTasks and NameSpaces in robot.yaml.
// Not every field is used in every case.
type TaskSettings struct {
	Name          string            `yaml:"Name"`          // Name of the task
	Path          string            `yaml:"Path"`          // Path to the executable or script
	Description   string            `yaml:"Description"`   // Description of the task
	NameSpace     string            `yaml:"NameSpace"`     // Namespace for shared memory/parameters
	ParameterSets []string          `yaml:"ParameterSets"` // Sets of parameters for this task
	Disabled      bool              `yaml:"Disabled"`      // Indicates if the task is disabled
	Homed         bool              `yaml:"Homed"`         // Runs in home directory context if true
	Privileged    *bool             `yaml:"Privileged"`    // Indicates if the task requires elevated privileges
	Parameters    []robot.Parameter `yaml:"Parameters"`    // Fixed parameters for the task
}

// ScheduledTask items defined in robot.yaml, mostly for scheduled jobs
type ScheduledTask struct {
	Schedule string           `yaml:"Schedule"` // Timespec for https://godoc.org/github.com/robfig/cron
	TaskSpec `yaml:",inline"` // Inlines TaskSpec fields
}

// PluginHelp specifies keywords and help text for the bot help system
type PluginHelp struct {
	Keywords []string `yaml:"Keywords"` // Match words for 'help XXX'
	Helptext []string `yaml:"Helptext"` // Help string for the keywords, conventionally starting with (bot) for commands or (hear) for general messages
}

// InputMatcher specifies the command or message to match for a plugin
type InputMatcher struct {
	Regex       string         `yaml:"Regex"`       // The regular expression string to match - bot adds ^\w* & \w*$
	Command     string         `yaml:"Command"`     // The name of the command to pass to the plugin with its arguments
	Label       string         `yaml:"Label"`       // ReplyMatchers use "Label" instead of "Command"
	ChannelOnly bool           `yaml:"ChannelOnly"` // Whether this matcher only applies in the main channel (not a thread)
	Contexts    []string       `yaml:"Contexts"`    // Labels for capture groups, for supporting "it" & optional args
	re          *regexp.Regexp `yaml:"-"`           // The compiled regular expression, logged if compilation fails
}

// JobTrigger specifies a user and message to trigger a job
type JobTrigger struct {
	Regex   string         `yaml:"Regex"`   // The regular expression string to match - bot adds ^\w* & \w*$
	User    string         `yaml:"User"`    // Required user to trigger this job, typically a webhook or integration
	Channel string         `yaml:"Channel"` // Required channel for the trigger
	re      *regexp.Regexp `yaml:"-"`       // The compiled regular expression, logged if compilation fails
}

// ParameterSet just stores a name, description, and parameters - they cannot be run.
type ParameterSet struct {
	name        string            `yaml:"-"`           // Name of the shared namespace
	Description string            `yaml:"Description"` // Optional description of the shared namespace
	Parameters  []robot.Parameter `yaml:"Parameters"`  // Parameters for the shared namespace
}

// Task configuration is common to tasks, plugins, or jobs. Any task, plugin, or job can call bot methods.
// Tasks are only defined in robot.yaml, and no external configuration is read in.
type Task struct {
	name          string            `yaml:"-"`               // Name of job or plugin; unique by type, but job & plugin can share
	taskType      taskType          `yaml:"-"`               // TaskGo or taskExternal
	Path          string            `yaml:"Path"`            // Path to the external executable for external scripts
	NameSpace     string            `yaml:"NameSpace"`       // Callers that share namespace share long-term memories and environment vars; defaults to name if not otherwise set
	Parameters    []robot.Parameter `yaml:"Parameters"`      // Fixed parameters for a given job; many jobs will use the same script with differing parameters
	ParameterSets []string          `yaml:"ParameterSets"`   //
	Description   string            `yaml:"Description"`     // Description of job or plugin
	AllowDirect   bool              `yaml:"AllowDirect"`     // Set this true if this plugin can be accessed via direct message
	DirectOnly    bool              `yaml:"DirectOnly"`      // Set this true if this plugin ONLY accepts direct messages
	Channel       string            `yaml:"Channel"`         // Channel where a job can be interacted with, or a scheduled task (job or plugin) runs
	Channels      []string          `yaml:"Channels"`        // Plugins only; Channels where the plugin is available. If empty, uses DefaultChannels
	AllChannels   bool              `yaml:"AllChannels"`     // If the Channels list is empty and AllChannels is true, the plugin should be active in all channels the bot is in
	RequireAdmin  bool              `yaml:"RequireAdmin"`    // Set to only allow administrators to access a plugin / run job
	Users         []string          `yaml:"Users"`           // If non-empty, list of all users with access to this plugin
	Elevator      string            `yaml:"Elevator"`        // Use an elevator other than the DefaultElevator
	Authorizer    string            `yaml:"Authorizer"`      // A plugin to call for authorizing users, should handle groups, etc.
	AuthRequire   string            `yaml:"AuthRequire"`     // An optional group/role name to be passed to the Authorizer plugin for group/role-based authorization
	ReplyMatchers []InputMatcher    `yaml:"ReplyMatchers"`   // Store this here for prompt*reply methods
	Config        interface{}       `yaml:"Config"`          // Arbitrary Plugin configuration, will be stored and provided in a thread-safe manner via GetTaskConfig()
	config        interface{}       `yaml:"ConfigInterface"` // A pointer to an empty struct that the bot can Unmarshal custom configuration into
	Disabled      bool              `yaml:"Disabled"`
	reason        string            `yaml:"-"`          // Why this job/plugin is disabled
	Privileged    bool              `yaml:"Privileged"` // Privileged jobs/plugins run with the privileged UID, privileged tasks require privileged pipelines
	Homed         bool              `yaml:"Homed"`      // Homed jobs/plugins start the pipeline with c.basePath = ".", homed tasks always run in "."
}

// Job - configuration only applicable to jobs. Read in from conf/jobs/<job>.yaml, which can also include anything from a Task.
type Job struct {
	Quiet     bool           `yaml:"Quiet"`     // Whether to quash "job started/ended" messages
	KeepLogs  int            `yaml:"KeepLogs"`  // How many runs of this job/plugin to keep history for
	Triggers  []JobTrigger   `yaml:"Triggers"`  // User/regex that triggers a job, e.g., a git-activated webhook or integration
	Arguments []InputMatcher `yaml:"Arguments"` // List of arguments to prompt the user for
	*Task     `yaml:",inline"`
}

// Plugin specifies the structure of a plugin configuration. Plugins should include an example/default config.
// Custom plugin configuration will be loaded from conf/plugins/<plugin>.yaml, which can also include anything from a Task.
type Plugin struct {
	AdminCommands            []string       `yaml:"AdminCommands"`            // A list of commands only a bot admin can use
	ElevatedCommands         []string       `yaml:"ElevatedCommands"`         // Commands that require elevation, usually via 2FA
	ElevateImmediateCommands []string       `yaml:"ElevateImmediateCommands"` // Commands that always require elevation prompting, regardless of timeouts
	AuthorizedCommands       []string       `yaml:"AuthorizedCommands"`       // Which commands to authorize
	AllowedHiddenCommands    []string       `yaml:"AllowedHiddenCommands"`    // Which commands are allowed to be hidden
	AuthorizeAllCommands     bool           `yaml:"AuthorizeAllCommands"`     // When ALL commands need to be authorized
	Help                     []PluginHelp   `yaml:"Help"`                     // All the keyword sets/help texts for this plugin
	CommandMatchers          []InputMatcher `yaml:"CommandMatchers"`          // Input matchers for messages that need to be directed to the bot
	MessageMatchers          []InputMatcher `yaml:"MessageMatchers"`          // Input matchers for messages the bot hears even when it’s not being spoken to
	AmbientMatchCommand      bool           `yaml:"AmbientMatchCommand"`      // Whether message matchers should also match when isCommand is true
	CatchAll                 bool           `yaml:"CatchAll"`                 // Plugins with CatchAll=true get called with command="catchall" and argument=<full message text to robot>
	MatchUnlisted            bool           `yaml:"MatchUnlisted"`            // Set to true if ambient message matches should be checked for users not listed in the UserRoster
	*Task                    `yaml:",inline"`
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
				Incoming:      &robot.ConnectorMessage{},
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
