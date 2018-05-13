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
	p       []*botPlugin
	nameMap map[string]int
	idMap   map[string]int
	sync.RWMutex
}

type externalScript struct {
	Name, Path string // List of names and paths for external plugins; relative paths are searched first in installpath, then configpath
}

var currentPlugins = &pluginList{
	[]*botPlugin{},
	nil,
	nil,
	sync.RWMutex{},
}

func (pl *pluginList) getPluginByName(name string) *botPlugin {
	pl.RLock()
	pi, ok := pl.nameMap[name]
	if !ok {
		Log(Error, fmt.Sprintf("Plugin '%s' not found calling getPluginByName", name))
		pl.RUnlock()
		return nil
	}
	plugin := pl.p[pi]
	pl.RUnlock()
	return plugin
}

func (pl *pluginList) getPluginByID(id string) *botPlugin {
	pl.RLock()
	pi, ok := pl.idMap[id]
	if !ok {
		Log(Error, fmt.Sprintf("Plugin '%s' not found calling getPluginByID", id))
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
)

// Plugin specifies the structure of a plugin configuration - plugins should include an example / default config
type botPlugin struct {
	pluginType               plugType        // plugGo, plugExternal - determines how commands are routed
	pluginPath               string          // Path to the external executable that expects <channel> <user> <command> <arg> <arg> from regex matches - for Plugtype=plugExternal only
	AllowDirect              bool            // Set this true if this plugin can be accessed via direct message
	DirectOnly               bool            // Set this true if this plugin ONLY accepts direct messages
	Channels                 []string        // Channels where the plugin is active - rifraf like "memes" should probably only be in random, but it's configurable. If empty uses DefaultChannels
	AllChannels              bool            // If the Channels list is empty and AllChannels is true, the plugin should be active in all the channels the bot is in
	RequireAdmin             bool            // Set to only allow administrators to access a plugin
	AdminCommands            []string        // A list of commands only a bot admin can use
	Elevator                 string          // Use an elevator other than the DefaultElevator
	ElevatedCommands         []string        // Commands that require elevation, usually via 2fa
	ElevateImmediateCommands []string        // Commands that always require elevation promting, regardless of timeouts
	Users                    []string        // If non-empty, list of all the users with access to this plugin
	Authorizer               string          // a plugin to call for authorizing users, should handle groups, etc.
	AuthRequire              string          // an optional group/role name to be passed to the Authorizer plugin, for group/role-based authorization determination
	AuthorizedCommands       []string        // Which commands to authorize
	AuthorizeAllCommands     bool            // when ALL commands need to be authorized
	Help                     []PluginHelp    // All the keyword sets / help texts for this plugin
	CommandMatchers          []InputMatcher  // Input matchers for messages that need to be directed to the 'bot
	MessageMatchers          []InputMatcher  // Input matchers for messages the 'bot hears even when it's not being spoken to
	CatchAll                 bool            // Whenever the robot is spoken to, but no plugin matches, plugins with CatchAll=true get called with command="catchall" and argument=<full text of message to robot>
	Config                   json.RawMessage // Arbitrary Plugin configuration, will be stored and provided in a thread-safe manner via GetPluginConfig()
	config                   interface{}     // A pointer to an empty struct that the bot can Unmarshal custom configuration into
	botCaller
}

// PluginHandler is the struct a plugin registers for the Gopherbot plugin API.
type PluginHandler struct {
	DefaultConfig string /* A yaml-formatted multiline string defining the default Plugin configuration. It should be liberally commented for use in generating
	custom configuration for the plugin. If a Config: section is defined, it should match the structure of the optional Config interface{} */
	Handler func(bot *Robot, command string, args ...string) PlugRetVal // The callback function called by the robot whenever a Command is matched
	Config  interface{}                                                 // An optional empty struct defining custom configuration for the plugin
}

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
			if plugin.Disabled {
				continue
			}
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

// RegisterPlugin allows plugins to register a PluginHandler in a func init().
// When the bot initializes, it will call each plugin's handler with a command
// "init", empty channel, the bot's username, and no arguments, so the plugin
// can store this information for, e.g., scheduled jobs.
// See builtins.go for the pluginHandlers definition.
func RegisterPlugin(name string, plug PluginHandler) {
	if stopRegistrations {
		return
	}
	if !pNameRe.MatchString(name) {
		log.Fatalf("Plugin name '%s' doesn't match plugin name regex '%s'", name, pNameRe.String())
	}
	if _, exists := pluginHandlers[name]; exists {
		log.Fatalf("Attempted plugin name registration duplicates builtIn or other Go plugin: %s", name)
	}
	pluginHandlers[name] = plug
}

func getPlugID(plug string) string {
	plugNameIDmap.Lock()
	plugID, ok := plugNameIDmap.m[plug]
	if ok {
		plugNameIDmap.Unlock()
		return plugID
	} else {
		// Generate a random id
		p := make([]byte, 16)
		rand.Read(p)
		plugID = fmt.Sprintf("%x", p)
		plugNameIDmap.m[plug] = plugID
		plugNameIDmap.Unlock()
		return plugID
	}
}

// loadPluginConfig() loads the configuration for all the plugins from
// /plugins/<pluginname>.yaml, assigns a callerID, and
// stores the resulting array in b.plugins. Bad plugins are skipped and logged.
// Plugin configuration is initially loaded into temporary data structures,
// then stored in the bot package under the global bot lock.
func (r *Robot) loadPluginConfig() {
	plugIndexByID := make(map[string]int)
	plugIndexByName := make(map[string]int)
	plist := make([]*botPlugin, 0, 14)

	// Copy some data from the bot under read lock, including external plugins
	robot.RLock()
	defaultAllowDirect := robot.defaultAllowDirect
	// copy the list of default channels
	pchan := make([]string, 0, len(robot.plugChannels))
	pchan = append(pchan, robot.plugChannels...)
	externalPlugins := make([]externalScript, 0, len(robot.externalPlugins))
	externalPlugins = append(externalPlugins, robot.externalPlugins...)
	robot.RUnlock() // we're done with bot data 'til the end

	i := 0

	for plugname := range pluginHandlers {
		plugin := &botPlugin{
			pluginType: plugGo,
			botCaller: botCaller {
				name: plugname,
				callerID: getPlugID(plugname),
			},
		}
		plist = append(plist, plugin)
		plugIndexByID[plugin.callerID] = i
		plugIndexByName[plugin.name] = i
		i++
	}

	for index, plug := range externalPlugins {
		if !pNameRe.MatchString(plug.Name) {
			Log(Error, fmt.Sprintf("Plugin name: '%s', index: %d doesn't match plugin name regex '%s', skipping", plug.Name, index+1, pNameRe.String()))
			continue
		}
		if dup, ok := plugIndexByName[plug.Name]; ok {
			msg := fmt.Sprintf("External plugin index: #%d, name: '%s' duplicates name of builtIn or Go plugin, skipping", index, plug.Name)
			Log(Error, msg)
			r.debug(plist[dup].callerID, msg, false)
			continue
		}
		plugin := &botPlugin{
			pluginPath: plug.Path,
			pluginType: plugExternal,
			botCaller: botCaller{
				name: plug.Name,
				callerID: getPlugID(plug.Name),
			},
		}
		plist = append(plist, plugin)
		plugIndexByID[plugin.callerID] = i
		plugIndexByName[plugin.name] = i
		i++
		if len(plug.Path) == 0 {
			msg := fmt.Sprintf("Plugin '%s' has zero-length path, disabling", plug.Name)
			Log(Error, msg)
			r.debug(plugin.callerID, msg, false)
			plugin.Disabled = true
			plugin.reason = msg
		}
	}

	// Load configuration for all valid plugins. Note that this is all being loaded
	// in to non-shared data structures that will replace current configuration
	// under lock at the end.
PlugLoop:
	for i, plugin := range plist {
		if plugin.Disabled {
			continue
		}
		pcfgload := make(map[string]json.RawMessage)
		Log(Debug, fmt.Sprintf("Loading configuration for plugin #%d - %s, type %d", i, plugin.name, plugin.pluginType))

		if plugin.pluginType == plugExternal {
			// External plugins spit their default config to stdout when called with command="configure"
			cfg, err := getExtDefCfg(plugin)
			if err != nil {
				msg := fmt.Sprintf("Error getting default configuration for external plugin, disabling: %v", err)
				Log(Error, msg)
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue
			}
			if len(*cfg) > 0 {
				r.debug(plugin.callerID, fmt.Sprintf("Loaded default config from the plugin, size: %d", len(*cfg)), false)
			} else {
				r.debug(plugin.callerID, "Unable to obtain default config from plugin, command 'configure' returned no content", false)
			}
			if err := yaml.Unmarshal(*cfg, &pcfgload); err != nil {
				msg := fmt.Sprintf("Error unmarshalling default configuration, disabling: %v", err)
				Log(Error, fmt.Errorf("Problem unmarshalling plugin default config for '%s', disabling: %v", plugin.name, err))
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue
			}
		} else {
			if err := yaml.Unmarshal([]byte(pluginHandlers[plugin.name].DefaultConfig), &pcfgload); err != nil {
				msg := fmt.Sprintf("Error unmarshalling default configuration, disabling: %v", err)
				Log(Error, fmt.Errorf("Problem unmarshalling plugin default config for '%s', disabling: %v", plugin.name, err))
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue
			}
		}
		// getConfigFile overlays the default config with configuration from the install path, then config path
		if err := r.getConfigFile("plugins/"+plugin.name+".yaml", plugin.callerID, false, pcfgload); err != nil {
			msg := fmt.Sprintf("Problem loading configuration file(s) for plugin '%s', disabling: %v", plugin.name, err)
			Log(Error, msg)
			r.debug(plugin.callerID, msg, false)
			plugin.Disabled = true
			plugin.reason = msg
			continue
		}
		if disjson, ok := pcfgload["Disabled"]; ok {
			disabled := false
			if err := json.Unmarshal(disjson, &disabled); err != nil {
				msg := fmt.Sprintf("Problem unmarshalling value for 'Disabled' in plugin '%s', disabling: %v", plugin.name, err)
				Log(Error, msg)
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue
			}
			if disabled {
				msg := fmt.Sprintf("Plugin '%s' is disabled by configuration", plugin.name)
				Log(Info, msg)
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue
			}
		}
		// Boolean false values can be explicitly false, or default to false
		// when not specified. In some cases that matters.
		explicitAllChannels := false
		explicitAllowDirect := false
		explicitDenyDirect := false
		denyDirect := false

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
				msg := fmt.Sprintf("Invalid configuration key for plugin '%s': %s - disabling", plugin.name, key)
				Log(Error, msg)
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue PlugLoop
			}

			if !skip {
				if err := json.Unmarshal(value, val); err != nil {
					msg := fmt.Sprintf("Disabling plugin '%s' - error unmarshalling value '%s': %v", plugin.name, key, err)
					Log(Error, msg)
					r.debug(plugin.callerID, msg, false)
					plugin.Disabled = true
					plugin.reason = msg
					continue PlugLoop
				}
			}

			switch key {
			case "AllowDirect":
				plugin.AllowDirect = *(val.(*bool))
				explicitAllowDirect = true
			case "DirectOnly":
				plugin.DirectOnly = *(val.(*bool))
			case "DenyDirect":
				denyDirect = *(val.(*bool))
				explicitDenyDirect = true
			case "Channels":
				plugin.Channels = *(val.(*[]string))
			case "AllChannels":
				plugin.AllChannels = *(val.(*bool))
				explicitAllChannels = true
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
		// End of reading configuration keys

		// Start sanity checking of configuration

		if plugin.DirectOnly {
			if explicitAllowDirect {
				if !plugin.AllowDirect {
					msg := fmt.Sprintf("Plugin '%s' has conflicting values for AllowDirect (false) and DirectOnly (true), disabling", plugin.name)
					Log(Error, msg)
					r.debug(plugin.callerID, msg, false)
					plugin.Disabled = true
					plugin.reason = msg
					continue
				}
			} else {
				Log(Debug, "DirectOnly specified without AllowDirect; setting AllowDirect = true")
				plugin.AllowDirect = true
				explicitAllowDirect = true
			}
		}

		if explicitAllowDirect && explicitDenyDirect && (plugin.AllowDirect == denyDirect) {
			msg := fmt.Sprintf("Plugin '%s' has conflicting values for AllowDirect and deprecated DenyDirect, disabling", plugin.name)
			Log(Error, msg)
			r.debug(plugin.callerID, msg, false)
			plugin.Disabled = true
			plugin.reason = msg
			continue
		}

		if explicitDenyDirect && !explicitAllowDirect {
			Log(Debug, "Deprecated DenyDirect specified without AllowDirect; setting AllowDirect = !DenyDirect")
			plugin.AllowDirect = !denyDirect
			explicitAllowDirect = true
		}

		if !explicitAllowDirect {
			plugin.AllowDirect = defaultAllowDirect
		}

		// Use bot default plugin channels if none defined, unless AllChannels requested.
		if len(plugin.Channels) == 0 {
			if len(pchan) > 0 {
				if !plugin.AllChannels { // AllChannels = true is always explicit
					plugin.Channels = pchan
				}
			} else { // no default channels specified
				if !explicitAllChannels { // if AllChannels wasn't explicitly configured, and no default channels, default to AllChannels = true
					plugin.AllChannels = true
				}
			}
		}
		// Note: you can't combine the channel length checking logic, the above
		// can change it.

		// Considering possible default channels, is the plugin visible anywhere?
		if len(plugin.Channels) > 0 {
			msg := fmt.Sprintf("Plugin '%s' will be active in channels %q", plugin.name, plugin.Channels)
			Log(Info, msg)
			r.debug(plugin.callerID, msg, false)
		} else {
			if !(plugin.AllowDirect || plugin.AllChannels) {
				msg := fmt.Sprintf("Plugin '%s' not visible in any channels or by direct message, disabling", plugin.name)
				Log(Error, msg)
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue
			} else {
				msg := fmt.Sprintf("Plugin '%s' has no channel restrictions configured; all channels: %t", plugin.name, plugin.AllChannels)
				Log(Info, msg)
				r.debug(plugin.callerID, msg, false)
			}
		}

		// Compile the regex's
		for i := range plugin.CommandMatchers {
			command := &plugin.CommandMatchers[i]
			regex := massageRegexp(command.Regex)
			re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
			if err != nil {
				msg := fmt.Sprintf("Disabling %s, couldn't compile command regular expression '%s': %v", plugin.name, regex, err)
				Log(Error, msg)
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue PlugLoop
			} else {
				command.re = re
			}
		}
		for i := range plugin.ReplyMatchers {
			reply := &plugin.ReplyMatchers[i]
			regex := massageRegexp(reply.Regex)
			re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
			if err != nil {
				msg := fmt.Sprintf("Skipping %s, couldn't compile reply regular expression '%s': %v", plugin.name, regex, err)
				Log(Error, msg)
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue PlugLoop
			} else {
				reply.re = re
			}
		}
		for i := range plugin.MessageMatchers {
			// Note that full message regexes don't get the beginning and end anchors added - the individual plugin
			// will need to do this if necessary.
			message := &plugin.MessageMatchers[i]
			regex := massageRegexp(message.Regex)
			re, err := regexp.Compile(regex)
			if err != nil {
				msg := fmt.Sprintf("Skipping %s, couldn't compile message regular expression '%s': %v", plugin.name, regex, err)
				Log(Error, msg)
				r.debug(plugin.callerID, msg, false)
				plugin.Disabled = true
				plugin.reason = msg
				continue PlugLoop
			} else {
				message.re = re
			}
		}

		// Make sure all security-related command lists resolve to actual
		// commands to guard against typos.
		cmdlist := []struct {
			ctype string
			clist []string
		}{
			{"elevated", plugin.ElevatedCommands},
			{"elevate immediate", plugin.ElevateImmediateCommands},
			{"authorized", plugin.AuthorizedCommands},
			{"admin", plugin.AdminCommands},
		}
		for _, cmd := range cmdlist {
			if len(cmd.clist) > 0 {
				for _, i := range cmd.clist {
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
						msg := fmt.Sprintf("Disabling %s, %s command %s didn't match a command from CommandMatchers or MessageMatchers", plugin.name, cmd.ctype, i)
						Log(Error, msg)
						r.debug(plugin.callerID, msg, false)
						plugin.Disabled = true
						plugin.reason = msg
						continue PlugLoop
					}
				}
			}
		}

		// For Go plugins, use the provided empty config struct to go ahead
		// and unmarshall Config. The GetPluginConfig call just sets a pointer
		// without unmshalling again.
		if plugin.pluginType == plugGo {
			// Copy the pointer to the empty config struct / empty struct (when no config)
			// pluginHandlers[name].Config is an empty struct for unmarshalling provided
			// in RegisterPlugin.
			pt := reflect.ValueOf(pluginHandlers[plugin.name].Config)
			if pt.Kind() == reflect.Ptr {
				if plugin.Config != nil {
					// reflect magic: create a pointer to a new empty config struct for the plugin
					plugin.config = reflect.New(reflect.Indirect(pt).Type()).Interface()
					if err := json.Unmarshal(plugin.Config, plugin.config); err != nil {
						msg := fmt.Sprintf("Error unmarshalling plugin config json to config, disabling: %v", err)
						Log(Error, msg)
						r.debug(plugin.callerID, msg, false)
						plugin.Disabled = true
						plugin.reason = msg
						continue
					}
				} else {
					// Providing custom config not required (should it be?)
					msg := fmt.Sprintf("Plugin '%s' has custom config, but none is configured", plugin.name)
					Log(Warn, msg)
					r.debug(plugin.callerID, msg, false)
				}
			} else {
				if plugin.Config != nil {
					msg := fmt.Sprintf("Custom configuration data provided for Go plugin '%s', but no config struct was registered; disabling", plugin.name)
					Log(Error, msg)
					r.debug(plugin.callerID, msg, false)
					plugin.Disabled = true
					plugin.reason = msg
				} else {
					Log(Debug, fmt.Sprintf("Config interface isn't a pointer, skipping unmarshal for Go plugin '%s'", plugin.name))
				}
			}
		}
		Log(Debug, fmt.Sprintf("Configured plugin #%d, '%s'", i, plugin.name))
	}
	// End of configuration loading. All invalid plugins are disabled.

	reInitPlugins := false
	currentPlugins.Lock()
	currentPlugins.p = plist
	currentPlugins.idMap = plugIndexByID
	currentPlugins.nameMap = plugIndexByName
	currentPlugins.Unlock()
	// loadPluginConfig is called in initBot, before the connector has started;
	// don't init plugins in that case.
	robot.RLock()
	if robot.Connector != nil {
		reInitPlugins = true
	}
	robot.RUnlock()
	if reInitPlugins {
		initializePlugins()
	}
}
