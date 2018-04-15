package bot

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
)

// PluginNames can be letters, numbers & underscores only, mainly so
// brain functions can use ':' as a separator.
var pNameRe = regexp.MustCompile(`[\w]+`)

// Global persistent map of plugin name to unique ID
var plugNameIDmap = struct {
	m map[string]string
	sync.Mutex
}{
	make(map[string]string),
	sync.Mutex{},
}

type pluginList struct {
	p       []*Plugin
	nameMap map[string]int
	idMap   map[string]int
	sync.RWMutex
}

var currentPlugins = &pluginList{
	[]*Plugin{},
	nil,
	nil,
	sync.RWMutex{},
}

func (pl *pluginList) getPluginByName(name string) *Plugin {
	pl.RLock()
	pi, ok := pl.nameMap[name]
	if !ok {
		Log(Error, fmt.Sprintf("Plugin \"%s\" not found calling getPluginByName", name))
		pl.RUnlock()
		return nil
	}
	plugin := pl.p[pi]
	pl.RUnlock()
	return plugin
}

func (pl *pluginList) getPluginByID(id string) *Plugin {
	pl.RLock()
	pi, ok := pl.idMap[id]
	if !ok {
		Log(Error, fmt.Sprintf("Plugin \"%s\" not found calling getPluginByID", id))
		pl.RUnlock()
		return nil
	}
	plugin := pl.p[pi]
	pl.RUnlock()
	return plugin
}

// PluginHelp specifies keywords and help text for the 'bot help system
type PluginHelp struct {
	Keywords []string // match words for 'help XXX'
	Helptext []string // help string to give for the keywords, conventionally starting with (bot) for commands or (hear) when the bot needn't be addressed directly
}

// InputMatcher specifies the command or message to match and what to pass to the plugin
type InputMatcher struct {
	Regex    string         // The regular expression string to match - bot adds ^\w* & \w*$
	Command  string         // The name of the command to pass to the plugin with it's arguments
	Label    string         // ReplyMatchers use "Label" instead of "Command"
	Contexts []string       // label the contexts corresponding to capture groups, for supporting "it" & optional args
	re       *regexp.Regexp // The compiled regular expression. If the regex doesn't compile, the 'bot will log an error
}

type plugType int

const (
	plugGo plugType = iota
	plugExternal
	plugBuiltin
)

// Plugin specifies the structure of a plugin configuration - plugins should include an example / default config
type Plugin struct {
	name                     string          // the name of the plugin, used as a key in to the
	pluginType               plugType        // plugGo, plugExternal, plugBuiltin - determines how commands are routed
	pluginPath               string          // Path to the external executable that expects <channel> <user> <command> <arg> <arg> from regex matches - for Plugtype=plugExternal only
	Disabled                 bool            // Set true to disable the plugin
	AllowDirect              bool            // Set this true if this plugin can be accessed via direct message
	DirectOnly               bool            // Set this true if this plugin ONLY accepts direct messages
	DenyDirect               bool            // Set true to override global DefaultAllowDirect = true
	Channels                 []string        // Channels where the plugin is active - rifraf like "memes" should probably only be in random, but it's configurable. If empty uses DefaultChannels
	AllChannels              bool            // If the Channels list is empty and AllChannels is true, the plugin should be active in all the channels the bot is in
	RequireAdmin             bool            // Set to only allow administrators to access a plugin
	AdminCommands            []string        // A list of commands only a bot admin can use
	Elevator                 string          // Use an elevator other than the DefaultElevator
	ElevatedCommands         []string        // Commands that require elevation, usually via 2fa
	ElevateImmediateCommands []string        // Commands that always require elevation promting, regardless of timeouts
	Users                    []string        // If non-empty, list of all the users with access to this plugin
	TrustedPlugins           []string        // list of plugins allowed to call this one
	Authorizer               string          // a plugin to call for authorizing users, should handle groups, etc.
	AuthRequire              string          // an optional group/role name to be passed to the Authorizer plugin, for group/role-based authorization determination
	AuthorizedCommands       []string        // Which commands to authorize
	AuthorizeAllCommands     bool            // when ALL commands need to be authorized
	Help                     []PluginHelp    // All the keyword sets / help texts for this plugin
	CommandMatchers          []InputMatcher  // Input matchers for messages that need to be directed to the 'bot
	ReplyMatchers            []InputMatcher  // Input matchers for replies to questions, only match after a RequestContinuation
	MessageMatchers          []InputMatcher  // Input matchers for messages the 'bot hears even when it's not being spoken to
	CatchAll                 bool            // Whenever the robot is spoken to, but no plugin matches, plugins with CatchAll=true get called with command="catchall" and argument=<full text of message to robot>
	Config                   json.RawMessage // Arbitrary Plugin configuration, will be stored and provided in a thread-safe manner via GetPluginConfig()
	config                   interface{}     // A pointer to an empty struct that the bot can Unmarshal custom configuration into
	pluginID                 string          // 32-char random ID for identifying plugins in callbacks
}

// PluginHandler is the struct a plugin registers for the Gopherbot plugin API.
type PluginHandler struct {
	DefaultConfig string /* A yaml-formatted multiline string defining the default Plugin configuration. It should be liberally commented for use in generating
	local/custom configuration for the plugin. If a Config: section is defined, it should match the structure of the optional Config interface{} */
	Handler func(bot *Robot, command string, args ...string) PlugRetVal // The callback function called by the robot whenever a Command is matched
	Config  interface{}                                                 // An optional empty struct defining custom configuration for the plugin
}

// pluginHandlers maps from plugin names to PluginV1 (later interface{} with a type selector, maybe)
var pluginHandlers = make(map[string]PluginHandler)

// stopRegistrations is set "true" when the bot is created to prevent registration outside of init functions
var stopRegistrations = false

// initialize sends the "init" command to every plugin
func initializePlugins() {
	currentPlugins.RLock()
	plugins := currentPlugins.p
	currentPlugins.RUnlock()
	robot.Lock()
	if !robot.shuttingDown {
		robot.Unlock()
		for _, plugin := range plugins {
			bot := &Robot{
				User:    robot.name,
				Channel: "",
				Format:  Variable,
			}
			Log(Info, "Initializing plugin:", plugin.name)
			go callPlugin(bot, plugin, false, false, "init")
		}
	} else {
		robot.Unlock()
	}
}

// Update passed-in regex so that a space can match a variable # of spaces
func massageRegexp(r string) string {
	replaceSpaceRe := regexp.MustCompile(`\[([^]]*) ([^]]*)\]`)
	regex := replaceSpaceRe.ReplaceAllString(r, `[$1\x20$2]`)
	regex = strings.Replace(regex, " ?", `\s*`, -1)
	regex = strings.Replace(regex, " ", `\s+`, -1)
	Log(Trace, fmt.Sprintf("Updated regex \"%s\" => \"%s\"", r, regex))
	return regex
}

// RegisterPlugin allows plugins to register a PluginHandler in a func init().
// When the bot initializes, it will call each plugin's handler with a command
// "init", empty channel, the bot's username, and no arguments, so the plugin
// can store this information for, e.g., scheduled jobs.
func RegisterPlugin(name string, plug PluginHandler) {
	if stopRegistrations {
		return
	}
	if _, exists := pluginHandlers[name]; exists {
		log.Fatal("Attempted registration of duplicate plugin name:", name)
	}
	pluginHandlers[name] = plug
}

// loadPluginConfig() loads the configuration for all the plugins from
// /plugins/<pluginname>.yaml, assigns a pluginID, and
// stores the resulting array in b.plugins. Bad plugins are skipped and logged.
// Plugin configuration is initially loaded into temporary data structures,
// then stored in the bot package under the global bot lock.
func (r *Robot) loadPluginConfig() {
	i := 0

	// Copy some data from the bot under lock
	robot.RLock()
	// Get a list of all plugins from the package pluginHandlers var and
	// the list of external plugins
	nump := len(pluginHandlers) + len(robot.externalPlugins)
	pnames := make([]string, nump)
	ptypes := make([]plugType, nump)
	eppaths := make(map[string]string) // Paths to external plugins
	plugIndexByID := make(map[string]int)
	plugIndexByName := make(map[string]int)
	pset := make(map[string]bool) // track plugin names

	defaultAllowDirect := robot.defaultAllowDirect

	// builtins come first so indexes match, see loop below
	// Note this doesn't need to be under RLock, but it needs to precede
	// external plugins. This should be fast enough that it doesn't matter.
	for _, plugin := range builtIns {
		pnames[i] = plugin
		pset[plugin] = true
		ptypes[i] = plugBuiltin
		i++
	}

	for index, plug := range robot.externalPlugins {
		if len(plug.Name) == 0 || len(plug.Path) == 0 {
			Log(Error, fmt.Sprintf("Skipping external plugin #%d with zero-length Name or Path", index+1))
			nump--
			continue
		}
		if !pNameRe.MatchString(plug.Name) {
			Log(Error, fmt.Sprintf("Plugin name \"%s\" doesn't match plugin name regex \"%s\", skipping", plug.Name, pNameRe.String()))
			nump--
			continue
		}
		if pset[plug.Name] {
			Log(Error, fmt.Sprintf("External plugin #%d, \"%s\" duplicates builtIn, skipping", index, plug.Name))
			nump--
			continue
		}
		pnames[i] = plug.Name
		pset[plug.Name] = true
		ptypes[i] = plugExternal
		eppaths[plug.Name] = plug.Path
		i++
	}
	// shrink slices when plugins were skipped
	pnames = pnames[0:nump]
	ptypes = ptypes[0:nump]
	// copy the list of default channels
	pchan := make([]string, 0, len(robot.plugChannels))
	pchan = append(pchan, robot.plugChannels...)
	robot.RUnlock() // we're done with bot data 'til the end

PlugHandlerLoop:
	for plug := range pluginHandlers {
		if !pNameRe.MatchString(plug) {
			Log(Error, fmt.Sprintf("Plugin name \"%s\" doesn't match plugin name regex \"%s\", skipping", plug, pNameRe.String()))
			nump--
			continue
		}
		if pset[plug] { // have to check builtIns, already loaded
			for _, plugin := range builtIns {
				if plug == plugin {
					continue PlugHandlerLoop // skip it, already loaded
				}
			}
			// Since external plugins can change on reload, just log an error if
			// we get a duplicate plugin name.
			Log(Error, "Plugin name duplicates external, skipping:", plug)
		} else {
			pnames[i] = plug
			ptypes[i] = plugGo
			i++
		}
	}
	// shrink slices when plugins were skipped
	pnames = pnames[0:nump]
	ptypes = ptypes[0:nump]
	plist := make([]*Plugin, 0, nump)

	// Because some plugins may be disabled, pnames and plugins won't necessarily sync
	plugIndex := 0

	var plugin *Plugin

PlugLoop:
	for i, plug := range pnames {
		plugNameIDmap.Lock()
		plugID, _ := plugNameIDmap.m[plug]
		plugNameIDmap.Unlock()

		plugin = new(Plugin)
		pcfgload := make(map[string]json.RawMessage)
		plugin.AllowDirect = defaultAllowDirect
		Log(Trace, fmt.Sprintf("Loading plugin #%d - %s, type %d", plugIndex, plug, ptypes[i]))

		plugin.pluginType = ptypes[i]
		if plugin.pluginType == plugExternal {
			// External plugins spit their default config to stdout when called with command="configure"
			plugin.pluginPath = eppaths[plug]
			cfg, err := getExtDefCfg(plugin)
			if err != nil {
				Log(Error, err)
				continue
			}
			if len(*cfg) > 0 {
				r.debug(plugID, fmt.Sprintf("Loaded default config from the plugin, size: %d", len(*cfg)), false)
			} else {
				r.debug(plugID, "Unable to obtain default config from plugin, command 'configure' returned no content", false)
			}
			if err := yaml.Unmarshal(*cfg, &pcfgload); err != nil {
				r.debug(plugID, fmt.Sprintf("Error unmarshalling default configuration: %v", err), false)
				Log(Error, fmt.Errorf("Problem unmarshalling plugin default config for \"%s\", skipping: %v", plug, err))
				continue
			}
		} else {
			if err := yaml.Unmarshal([]byte(pluginHandlers[plug].DefaultConfig), &pcfgload); err != nil {
				r.debug(plugID, fmt.Sprintf("Error unmarshalling default configuration: %v", err), false)
				Log(Error, fmt.Errorf("Problem unmarshalling plugin default config for \"%s\", skipping: %v", plug, err))
				continue
			}
		}
		// getConfigFile overlays the default config with local config
		if err := r.getConfigFile("plugins/"+plug+".yaml", plugID, false, pcfgload); err != nil {
			Log(Error, fmt.Errorf("Problem with local configuration for plugin \"%s\", skipping: %v", plug, err))
			continue
		}
		if disjson, ok := pcfgload["Disabled"]; ok {
			disabled := false
			if err := json.Unmarshal(disjson, &disabled); err != nil {
				Log(Error, fmt.Errorf("Problem reading value for \"Disabled\" in plugin \"%s\", skipping: %v", plug, err))
				continue
			}
			if disabled {
				Log(Info, fmt.Sprintf("Plugin \"%s\" is disabled, skipping", plug))
				continue
			}
		}
		for key, value := range pcfgload {
			var strval string
			var boolval bool
			var sarrval []string
			var hval []PluginHelp
			var mval []InputMatcher
			var val interface{}
			skip := false
			switch key {
			case "Elevator", "Authorizer", "AuthRequire":
				val = &strval
			case "Disabled", "AllowDirect", "DirectOnly", "DenyDirect", "AllChannels", "RequireAdmin", "AuthorizeAllCommands", "CatchAll":
				val = &boolval
			case "Channels", "ElevatedCommands", "ElevateImmediateCommands", "Users", "TrustedPlugins", "AuthorizedCommands", "AdminCommands":
				val = &sarrval
			case "Help":
				val = &hval
			case "CommandMatchers", "ReplyMatchers", "MessageMatchers":
				val = &mval
			case "Config":
				skip = true
			default:
				err := fmt.Errorf("Invalid configuration key for plugin \"%s\": %s - skipping", plug, key)
				Log(Error, err)
				continue PlugLoop
			}
			if !skip {
				if err := json.Unmarshal(value, val); err != nil {
					err = fmt.Errorf("Skipping plugin \"%s\" - error unmarshalling value \"%s\": %v", plug, key, err)
					Log(Error, err)
					continue PlugLoop
				}
			}
			switch key {
			case "AllowDirect":
				plugin.AllowDirect = *(val.(*bool))
			case "DirectOnly":
				plugin.DirectOnly = *(val.(*bool))
			case "DenyDirect":
				plugin.DenyDirect = *(val.(*bool))
			case "Channels":
				plugin.Channels = *(val.(*[]string))
			case "AllChannels":
				plugin.AllChannels = *(val.(*bool))
			case "RequireAdmin":
				plugin.RequireAdmin = *(val.(*bool))
			case "AdminCommands":
				plugin.AdminCommands = *(val.(*[]string))
			case "Elevator":
				plugin.Elevator = *(val.(*string))
			case "ElevatedCommands":
				plugin.ElevatedCommands = *(val.(*[]string))
			case "ElevateImmediateCommands":
				plugin.ElevateImmediateCommands = *(val.(*[]string))
			case "Users":
				plugin.Users = *(val.(*[]string))
			case "TrustedPlugins":
				plugin.TrustedPlugins = *(val.(*[]string))
			case "Authorizer":
				plugin.Authorizer = *(val.(*string))
			case "AuthRequire":
				plugin.AuthRequire = *(val.(*string))
			case "AuthorizedCommands":
				plugin.AuthorizedCommands = *(val.(*[]string))
			case "AuthorizeAllCommands":
				plugin.AuthorizeAllCommands = *(val.(*bool))
			case "Help":
				plugin.Help = *(val.(*[]PluginHelp))
			case "CommandMatchers":
				plugin.CommandMatchers = *(val.(*[]InputMatcher))
			case "ReplyMatchers":
				plugin.ReplyMatchers = *(val.(*[]InputMatcher))
			case "MessageMatchers":
				plugin.MessageMatchers = *(val.(*[]InputMatcher))
			case "CatchAll":
				plugin.CatchAll = *(val.(*bool))
			case "Config":
				plugin.Config = value
			}
		}
		Log(Info, "Loaded configuration for plugin", plug)
		// Use bot default plugin channels if none defined, unless AllChannels requested. Admin can override.
		if len(plugin.Channels) == 0 && len(pchan) > 0 && !plugin.AllChannels {
			plugin.Channels = pchan
		}
		Log(Info, fmt.Sprintf("Plugin \"%s\" will be active in channels %q; all channels: %t", plug, plugin.Channels, plugin.AllChannels))
		// Compile the regex's
		for i := range plugin.CommandMatchers {
			command := &plugin.CommandMatchers[i]
			regex := massageRegexp(command.Regex)
			re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
			if err != nil {
				Log(Error, fmt.Errorf("Skipping %s, couldn't compile command regular expression \"%s\": %v", plug, regex, err))
				continue PlugLoop
			}
			command.re = re
		}
		for i := range plugin.ReplyMatchers {
			reply := &plugin.ReplyMatchers[i]
			regex := massageRegexp(reply.Regex)
			re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
			if err != nil {
				Log(Error, fmt.Errorf("Skipping %s, couldn't compile reply regular expression \"%s\": %v", plug, regex, err))
				continue PlugLoop
			}
			reply.re = re
		}
		for i := range plugin.MessageMatchers {
			// Note that full message regexes don't get the beginning and end anchors added - the individual plugin
			// will need to do this if necessary.
			message := &plugin.MessageMatchers[i]
			regex := massageRegexp(message.Regex)
			re, err := regexp.Compile(regex)
			if err != nil {
				Log(Error, fmt.Errorf("Skipping %s, couldn't compile message regular expression \"%s\": %v", plug, regex, err))
				continue PlugLoop
			}
			message.re = re
		}
		if len(plugin.ElevatedCommands) > 0 {
			for _, i := range plugin.ElevatedCommands {
				cmdfound := false
				for _, j := range plugin.CommandMatchers {
					if i == j.Command {
						cmdfound = true
						break
					}
				}
				if !cmdfound {
					for _, j := range plugin.MessageMatchers {
						if i == j.Command {
							cmdfound = true
							break
						}
					}
				}
				if !cmdfound {
					Log(Error, fmt.Errorf("Skipping %s, elevated command %s didn't match a command from CommandMatchers or MessageMatchers", plug, i))
					continue PlugLoop
				}
			}
		}
		plugin.name = plug
		// Copy the pointer to the empty config struct / empty struct (when no config)
		pt := reflect.ValueOf(pluginHandlers[plug].Config)
		if pt.Kind() == reflect.Ptr {
			if plugin.Config != nil {
				// reflect magic: create a pointer to a new empty config struct for the plugin
				plugin.config = reflect.New(reflect.Indirect(pt).Type()).Interface()
				if err := json.Unmarshal(plugin.Config, plugin.config); err != nil {
					Log(Error, fmt.Sprintf("Error unmarshalling plugin config json to config: %v", err))
				}
			} else {
				Log(Debug, fmt.Sprintf("Plugin \"%s\" has custom config, but no local custom config provided", plug))
			}
		} else {
			Log(Debug, "config interface isn't a pointer, skipping unmarshal for plugin:", plug)
		}
		if plugID != "" {
			plugin.pluginID = plugID
		} else {
			plugNameIDmap.Lock()
			// Generate a random id
			p := make([]byte, 16)
			rand.Read(p)
			plugin.pluginID = fmt.Sprintf("%x", p)
			plugNameIDmap.m[plugin.name] = plugin.pluginID
			plugNameIDmap.Unlock()
		}
		plugIndexByID[plugin.pluginID] = plugIndex
		plugIndexByName[plugin.name] = plugIndex
		Log(Trace, fmt.Sprintf("Mapped plugin %s to ID %s", plugin.name, plugin.pluginID))
		// Store this plugin's config in the temporary list
		Log(Info, fmt.Sprintf("Recorded plugin #%d, \"%s\"", plugIndex, plugin.name))
		plist = append(plist, plugin)
		plugIndex++
	}

	reInitPlugins := false
	currentPlugins.Lock()
	currentPlugins.p = plist
	currentPlugins.idMap = plugIndexByID
	currentPlugins.nameMap = plugIndexByName
	currentPlugins.Unlock()
	// loadPluginConfig is called in initBot, before the connector has started;
	// don't init plugins in that case.
	robot.Lock()
	if robot.Connector != nil {
		reInitPlugins = true
	}
	robot.Unlock()
	if reInitPlugins {
		initializePlugins()
	}
}
