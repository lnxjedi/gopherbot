package bot

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
)

// Struct for ScheduledTasks (gopherbot.yaml) and AddTask (robot method)
type taskSpec struct {
	Name      string   // name of the job or plugin
	Arguments []string // for plugins only
	// environment vars for jobs and plugins, unused in AddTask, which should
	// make calls to SetParameter()
	Parameters []parameter
}

// PluginNames can be letters, numbers & underscores only, mainly so
// brain functions can use ':' as a separator.
var taskNameRe = regexp.MustCompile(`[\w]+`)

// Global persistent map of plugin name to unique ID
var taskNameIDmap = struct {
	m map[string]string
	sync.Mutex
}{
	make(map[string]string),
	sync.Mutex{},
}

type taskList struct {
	t       []interface{}
	nameMap map[string]int
	idMap   map[string]int
	sync.RWMutex
}

type externalScript struct {
	// List of names, paths and types for external plugins and jobs; relative paths are searched first in installpath, then configpath
	Name, Path, Type string
}

var currentTasks = &taskList{
	nil,
	nil,
	nil,
	sync.RWMutex{},
}

func getTask(t interface{}) (*botTask, *botPlugin, *botJob) {
	p, ok := t.(*botPlugin)
	if ok {
		return p.botTask, p, nil
	} else {
		j := t.(*botJob)
		return j.botTask, nil, j
	}
}

func (tl *taskList) getTaskByName(name string) (*botTask, *botPlugin, *botJob) {
	tl.RLock()
	ti, ok := tl.nameMap[name]
	if !ok {
		Log(Error, fmt.Sprintf("Task '%s' not found calling getTaskByName", name))
		tl.RUnlock()
		return nil, nil, nil
	}
	task := tl.t[ti]
	tl.RUnlock()
	return getTask(task)
}

func (tl *taskList) getTaskByID(id string) (*botTask, *botPlugin, *botJob) {
	tl.RLock()
	ti, ok := tl.idMap[id]
	if !ok {
		Log(Error, fmt.Sprintf("Task '%s' not found calling getTaskByID", id))
		tl.RUnlock()
		return nil, nil, nil
	}
	task := tl.t[ti]
	tl.RUnlock()
	return getTask(task)
}

func getPlugin(t interface{}) *botPlugin {
	p, ok := t.(*botPlugin)
	if ok {
		return p
	}
	return nil
}

func getJob(t interface{}) *botJob {
	j, ok := t.(*botJob)
	if ok {
		return j
	}
	return nil
}

// parameters are provided to jobs and plugins as environment variables
type parameter struct {
	Name, Value string
}

// items in gopherbot.yaml
type scheduledTask struct {
	Schedule string // timespec for https://godoc.org/github.com/robfig/cron
	taskSpec
}

// PluginHelp specifies keywords and help text for the 'bot help system
type PluginHelp struct {
	Keywords []string // match words for 'help XXX'
	Helptext []string // help string to give for the keywords, conventionally starting with (bot) for commands or (hear) when the bot needn't be addressed directly
}

type matcherType int

const (
	plugCommands matcherType = iota
	plugMessages
	jobTriggers
	runJob
)

// InputMatcher specifies the command or message to match for a plugin, or user and message to trigger a job
type InputMatcher struct {
	Regex       string         // The regular expression string to match - bot adds ^\w* & \w*$
	Command     string         // The name of the command to pass to the plugin with it's arguments
	Label       string         // ReplyMatchers use "Label" instead of "Command"
	Contexts    []string       // label the contexts corresponding to capture groups, for supporting "it" & optional args
	User        string         // jobs only; user that can trigger this job, normally git-activated webhook or integration
	Parameters  []string       // jobs only; names of parameters (environment vars) where regex matches are stored, in order of capture groups
	re          *regexp.Regexp // The compiled regular expression. If the regex doesn't compile, the 'bot will log an error
	matcherType matcherType    // What kind of message matched
}

type plugType int

const (
	plugGo plugType = iota
	plugExternal
)

// a botTask can be a plugin or a job, both capable of calling Robot methods
type botTask struct {
	name          string          // name of job or plugin; unique by type, but job & plugin can share
	scriptPath    string          // Path to the external executable for jobs or Plugtype=plugExternal only
	NameSpace     string          // callers that share namespace share long-term memories and environment vars; defaults to name if not otherwise set
	Description   string          // description of job or plugin
	MaxHistories  int             // how many runs of this job/plugin to keep history for
	AllowDirect   bool            // Set this true if this plugin can be accessed via direct message
	DirectOnly    bool            // Set this true if this plugin ONLY accepts direct messages
	Channels      []string        // Channels where the task is available - rifraf like "memes" should probably only be in random, but it's configurable. If empty uses DefaultChannels
	AllChannels   bool            // If the Channels list is empty and AllChannels is true, the plugin should be active in all the channels the bot is in
	RequireAdmin  bool            // Set to only allow administrators to access a plugin
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

// stuff read in conf/jobs/<job>.yaml
type botJob struct {
	Channel            string         // where job status updates are posted
	Notify             string         // user to notify on failure; job runs with this User for Replies
	Verbose            bool           // whether to send verbose "job started/ended" messages
	Triggers           []InputMatcher // user/regex that triggers a job, e.g. a git-activated webhook or integration
	RequiredParameters []string       // required in schedule, prompted to user for interactive
	*botTask
}

// Plugin specifies the structure of a plugin configuration - plugins should include an example / default config
type botPlugin struct {
	pluginType               plugType       // plugGo, plugExternal - determines how commands are routed
	AdminCommands            []string       // A list of commands only a bot admin can use
	ElevatedCommands         []string       // Commands that require elevation, usually via 2fa
	ElevateImmediateCommands []string       // Commands that always require elevation promting, regardless of timeouts
	AuthorizedCommands       []string       // Which commands to authorize
	AuthorizeAllCommands     bool           // when ALL commands need to be authorized
	Help                     []PluginHelp   // All the keyword sets / help texts for this plugin
	CommandMatchers          []InputMatcher // Input matchers for messages that need to be directed to the 'bot
	MessageMatchers          []InputMatcher // Input matchers for messages the 'bot hears even when it's not being spoken to
	CatchAll                 bool           // Whenever the robot is spoken to, but no plugin matches, plugins with CatchAll=true get called with command="catchall" and argument=<full text of message to robot>
	*botTask
}

// PluginHandler is the struct a plugin registers for the Gopherbot plugin API.
type PluginHandler struct {
	DefaultConfig string /* A yaml-formatted multiline string defining the default Plugin configuration. It should be liberally commented for use in generating
	custom configuration for the plugin. If a Config: section is defined, it should match the structure of the optional Config interface{} */
	Handler func(bot *Robot, command string, args ...string) TaskRetVal // The callback function called by the robot whenever a Command is matched
	Config  interface{}                                                 // An optional empty struct defining custom configuration for the plugin
}

var pluginHandlers = make(map[string]PluginHandler)

// stopRegistrations is set "true" when the bot is created to prevent registration outside of init functions
var stopRegistrations = false

// initialize sends the "init" command to every plugin
func initializePlugins() {
	currentTasks.RLock()
	tasks := taskList{
		currentTasks.t,
		currentTasks.nameMap,
		currentTasks.idMap,
		sync.RWMutex{},
	}
	currentTasks.RUnlock()
	bot := &botContext{
		tasks: tasks,
	}
	bot.registerActive()
	robot.Lock()
	if !robot.shuttingDown {
		robot.Unlock()
		for _, t := range tasks.t {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			if task.Disabled {
				continue
			}
			Log(Info, "Initializing plugin:", task.name)
			bot.callTask(t, "init")
		}
	} else {
		robot.Unlock()
	}
	bot.deregister()
}

// Update passed-in regex so that a space can match a variable # of spaces,
// to prevent cut-n-paste spacing related non-matches.
func massageRegexp(r string) string {
	replaceSpaceRe := regexp.MustCompile(`\[([^]]*) ([^]]*)\]`)
	regex := replaceSpaceRe.ReplaceAllString(r, `[$1\x20$2]`)
	regex = strings.Replace(regex, " ?", `\s*`, -1)
	regex = strings.Replace(regex, " ", `\s+`, -1)
	Log(Trace, fmt.Sprintf("Updated regex '%s' => '%s'", r, regex))
	return regex
}

// RegisterPlugin allows Go plugins to register a PluginHandler in a func init().
// When the bot initializes, it will call each plugin's handler with a command
// "init", empty channel, the bot's username, and no arguments, so the plugin
// can store this information for, e.g., scheduled jobs.
// See builtins.go for the pluginHandlers definition.
func RegisterPlugin(name string, plug PluginHandler) {
	if stopRegistrations {
		return
	}
	if !taskNameRe.MatchString(name) {
		log.Fatalf("Plugin name '%s' doesn't match plugin name regex '%s'", name, taskNameRe.String())
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
