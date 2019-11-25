package bot

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
)

// Regex for task/job/plugin/NameSpace names. NOTE: if this changes,
// command regexes in jobbuiltins.go will need to be changed.
// Identifiers can be letters, numbers & underscores only, mainly so
// brain functions can use ':' as a separator.
const identifierRegex = `[A-Za-z][\w-]*`

var identifierRe = regexp.MustCompile(identifierRegex)

// Global persistent map of plugin name to unique ID
var taskNameIDmap = struct {
	m map[string]string
	sync.Mutex
}{
	make(map[string]string),
	sync.Mutex{},
}

type taskList struct {
	t          []interface{}
	nameMap    map[string]int
	idMap      map[string]int
	nameSpaces map[string]struct{}
	//	sync.RWMutex
}

var currentTasks = struct {
	*taskList
	sync.Mutex
}{
	&taskList{},
	sync.Mutex{},
}

func getTask(t interface{}) (*BotTask, *BotPlugin, *BotJob) {
	p, ok := t.(*BotPlugin)
	if ok {
		return p.BotTask, p, nil
	}
	j, ok := t.(*BotJob)
	if ok {
		return j.BotTask, nil, j
	}
	return t.(*BotTask), nil, nil
}

func (tl *taskList) getTaskByName(name string) interface{} {
	ti, ok := tl.nameMap[name]
	if !ok {
		Log(robot.Error, "task '%s' not found calling getTaskByName", name)
		return nil
	}
	task := tl.t[ti]
	return task
}

// TaskSpec is the structure for ScheduledJobs (gopherbot.yaml) and AddTask (robot method)
type TaskSpec struct {
	Name      string // name of the job or plugin
	Command   string // plugins only
	Arguments []string
	task      interface{} // populated in AddTask
}

// Parameter items are provided to jobs and plugins as environment variables
type Parameter struct {
	Name, Value string
}

// ExternalTask struct for ExternalPlugins, ExternalJobs and ExternalTasks in gopherbot.yaml.
// Note that this is the only configuration supplied for an ExternalTask.
type ExternalTask struct {
	Name, Path, Description, NameSpace string
	Disabled                           bool
	Parameters                         []Parameter
}

// LoadableModule struct for loading external modules.
type LoadableModule struct {
	Name, Path, Description, NameSpace string
	Disabled                           bool
}

// ScheduledTask items defined in gopherbot.yaml, mostly for scheduled jobs
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

// BotTask configuration is common to tasks, plugins or jobs. Any task, plugin or job can call bot methods. Note that tasks are only defined
// in gopherbot.yaml, and no external configuration is read in.
type BotTask struct {
	name          string          // name of job or plugin; unique by type, but job & plugin can share
	taskType      taskType        // taskGo or taskExternal
	Path          string          // Path to the external executable for jobs or Plugtype=taskExternal only
	NameSpace     string          // callers that share namespace share long-term memories and environment vars; defaults to name if not otherwise set
	Parameters    []Parameter     // Fixed parameters for a given job; many jobs will use the same script with differing parameters
	Description   string          // description of job or plugin
	AllowDirect   bool            // Set this true if this plugin can be accessed via direct message
	DirectOnly    bool            // Set this true if this plugin ONLY accepts direct messages
	Channel       string          // channel where a job can be interracted with, channel where a scheduled task (job or plugin) runs
	Channels      []string        // plugins only; Channels where the plugin is available - rifraf like "memes" should probably only be in random, but it's configurable. If empty uses DefaultChannels
	AllChannels   bool            // If the Channels list is empty and AllChannels is true, the plugin should be active in all the channels the bot is in
	RequireAdmin  bool            // Set to only allow administrators to access a plugin / run job
	Protected     bool            // Protected jobs run with wd = custom config directory; all other jobs run in workSpace
	Users         []string        // If non-empty, list of all the users with access to this plugin
	Elevator      string          // Use an elevator other than the DefaultElevator
	Authorizer    string          // a plugin to call for authorizing users, should handle groups, etc.
	AuthRequire   string          // an optional group/role name to be passed to the Authorizer plugin, for group/role-based authorization determination
	taskID        string          // 32-char random ID for identifying plugins/jobs
	ReplyMatchers []InputMatcher  // store this here for prompt*reply methods
	Config        json.RawMessage // Arbitrary Plugin configuration, will be stored and provided in a thread-safe manner via GetTaskConfig()
	config        interface{}     // A pointer to an empty struct that the bot can Unmarshal custom configuration into
	Disabled      bool
	reason        string // why this job/plugin is disabled
}

// BotJob - configuration only applicable to jobs. Read in from conf/jobs/<job>.yaml, which can also include anything from a BotTask.
type BotJob struct {
	Quiet       bool           // whether to quash "job started/ended" messages
	HistoryLogs int            // how many runs of this job/plugin to keep history for
	Triggers    []JobTrigger   // user/regex that triggers a job, e.g. a git-activated webhook or integration
	Arguments   []InputMatcher // list of arguments to prompt the user for
	*BotTask
}

// BotPlugin specifies the structure of a plugin configuration - plugins should include an example / default config. Custom plugin configuration
// will be loaded from conf/plugins/<plugin>.yaml, which can also include anything from a BotTask.
type BotPlugin struct {
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
	*BotTask
}

var pluginHandlers = make(map[string]robot.PluginHandler)

// stopRegistrations is set "true" when the bot is created to prevent registration outside of init functions
var stopRegistrations = false

// initialize sends the "init" command to every plugin
func initializePlugins() {
	currentTasks.Lock()
	tasks := taskList{
		currentTasks.t,
		currentTasks.nameMap,
		currentTasks.idMap,
		currentTasks.nameSpaces,
	}
	currentTasks.Unlock()
	c := &botContext{
		environment: make(map[string]string),
		tasks:       tasks,
	}
	c.registerActive(nil)
	botCfg.Lock()
	if !botCfg.shuttingDown {
		botCfg.Unlock()
		for _, t := range tasks.t {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			if task.Disabled {
				continue
			}
			Log(robot.Info, "Initializing plugin: %s", task.name)
			c.callTask(t, "init")
		}
	} else {
		botCfg.Unlock()
	}
	c.deregister()
}

// RegisterPlugin allows Go plugins to register a PluginHandler in a func init().
// When the bot initializes, it will call each plugin's handler with a command
// "init", empty channel, the bot's username, and no arguments, so the plugin
// can store this information for, e.g., scheduled jobs.
// See builtins.go for the pluginHandlers definition.
func RegisterPlugin(name string, plug robot.PluginHandler) {
	if stopRegistrations {
		return
	}
	if !identifierRe.MatchString(name) {
		log.Fatalf("Plugin name '%s' doesn't match plugin name regex '%s'", name, identifierRe.String())
	}
	if name == "bot" {
		log.Fatalf("Illegal Go plugin name registration for 'bot'")
	}
	if _, exists := pluginHandlers[name]; exists {
		log.Fatalf("Attempted plugin name registration duplicates builtIn or other Go plugin: %s", name)
	}
	pluginHandlers[name] = plug
}

func getTaskID(plug string) string {
	taskNameIDmap.Lock()
	taskID, ok := taskNameIDmap.m[plug]
	if ok {
		taskNameIDmap.Unlock()
		return taskID
	}
	// Generate a random id
	p := make([]byte, 16)
	rand.Read(p)
	taskID = fmt.Sprintf("%x", p)
	taskNameIDmap.m[plug] = taskID
	taskNameIDmap.Unlock()
	return taskID
}
