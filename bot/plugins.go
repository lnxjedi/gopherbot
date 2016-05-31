package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"sync"
)

// PluginNames can be letters, numbers & underscores only, mainly so
// brain functions can use ':' as a separator.
const pNameRegex = `[\w]+`

var pNameRe = regexp.MustCompile(pNameRegex)

// PluginHelp specifies keywords and help text for the 'bot help system
type PluginHelp struct {
	Keywords []string // match words for 'help XXX'
	Helptext []string // help string to give for the keywords, conventionally starting with (bot) for commands or (hear) when the bot needn't be addressed directly
}

// InputMatchers specify the command or message to match and what to pass to the plugin
type InputMatcher struct {
	Regex   string         // The regular expression string to match - bot adds ^\w* & \w*$
	Command string         // The name of the command to pass to the plugin with it's arguments
	re      *regexp.Regexp // The compiled regular expression. If the regex doesn't compile, the 'bot will log an error
}

type plugType int

const (
	plugGo plugType = iota
	plugExternal
	plugBuiltin
)

// Plugin specifies the structure of a plugin configuration - plugins should include an example / default config
type Plugin struct {
	Name           string          // the name of the plugin, used as a key in to the
	pluginType     plugType        // plugGo, plugExternal, plugBuiltin - determines how commands are routed
	PluginPath     string          // Path to the external executable that expects <channel> <user> <command> <arg> <arg> from regex matches - for Plugtype=shell only
	Disabled       bool            // Set true to disable the plugin
	DisallowDirect bool            // Set this true if this plugin can never be accessed via direct message
	Channels       []string        // Channels where the plugin is active - rifraf like "memes" should probably only be in random, but it's configurable. If empty uses DefaultChannels
	AllChannels    bool            // If the Channels list is empty and AllChannels is true, the plugin should be active in all the channels the bot is in
	Trusted        bool            // Administrator must set this true to allow the plugin to check OTP codes (to prevent a bad plugin from trying them all)
	Users          []string        // If non-empty, list of all the users with access to this plugin
	Help           []PluginHelp    // All the keyword sets / help texts for this plugin
	CommandMatches []InputMatcher  // Input matchers for messages that need to be directed to the 'bot
	ReplyMatchers  []InputMatcher  // Input matchers for replies to questions, only match after a RequestContinuation
	MessageMatches []InputMatcher  // Input matchers for messages the 'bot hears even when it's not being spoken to
	CatchAll       bool            // Whenever the robot is spoken to, but no plugin matches, plugins with CatchAll=true get called with command="catchall" and argument=<full text of message to robot>
	Config         json.RawMessage // Plugin Configuration, raw JSON that can be sent to a script plugin
	config         interface{}     // A pointer to a struct of type configType filled in with JSON-unmarshalled data from plugin.json Config, or an empty struct{} if no config needed
	pluginID       string          // 32-char random ID for identifying plugins in callbacks
	lock           sync.Mutex      // For use with the robot's Brain
}

// type PluginV1 is the struct a plugin registers for Gopherbot plugin API
// Version 1; (there may never actually be a V2+)
type PluginHandler struct {
	Config  interface{} // A pointer to an empty config struct for deserializing JSON into
	Handler func(bot Robot, command string, args ...string)
}

// pluginHandlers maps from plugin names to PluginV1 (later interface{} with a type selector, maybe)
var pluginHandlers map[string]PluginHandler = make(map[string]PluginHandler)

// stopRegistrations is set "true" when the bot is created to prevent registration outside of init functions
var stopRegistrations bool = false

// initialize sends the "init" command to every plugin
func (b *robot) initializePlugins() {
	b.lock.RLock()
	defer b.lock.RUnlock()
	bot := Robot{
		User:    b.name,
		Channel: "",
		Format:  Variable,
		robot:   b,
	}
	for _, plugin := range b.plugins {
		b.Log(Info, "Initializing plugin:", plugin.Name)
		go b.callPlugin(bot, plugin, "init")
	}
}

// RegisterPlugin allows plugins to register a handler function in a func init().
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
// $GOPHER_LOCALDIR/plugins/<pluginname>.json (excepting builtins), assigns
// a pluginID, and stores the resulting array in b.plugins. Bad plugins
// are skipped and logged.
// Plugin configuration is initially loaded into temporary data structures,
// then stored in the robot under the global bot lock.
func (b *robot) loadPluginConfig() {
	i := 0

	// Copy some data from the bot under lock
	b.lock.RLock()
	// Get a list of all plugins from the package pluginHandlers var and
	// the list of external plugins
	nump := len(pluginHandlers) + len(b.externalPlugins)
	pnames := make([]string, nump)
	ptypes := make([]plugType, nump)
	pfinder := make(map[string]int) // keep a map of pluginIDs to identify plugins during a callback
	pset := make(map[string]bool)   // track plugin names

	// builtins come first so indexes match, see loop below
	// Note this doesn't need to be under RLock, but it needs to precede
	// external plugins. This should be fast enough that it doesn't matter.
	for _, plugin := range builtIns {
		pnames[i] = plugin
		pset[plugin] = true
		ptypes[i] = plugBuiltin
		i++
	}

	for _, plug := range b.externalPlugins {
		if !pNameRe.MatchString(plug) {
			b.Log(Error, "Plugin name \"%s\" doesn't match plugin name regex \"%s\", skipping")
			continue
		}
		pnames[i] = plug
		if pset[plug] {
			b.Log(Error, "External plugin name duplicates builtIn, skipping:", plug)
			continue
		}
		pset[plug] = true
		ptypes[i] = plugExternal
		i++
	}
	// copy the list of default channels
	pchan := make([]string, 0, len(b.plugChannels))
	pchan = append(pchan, b.plugChannels...)
	b.lock.RUnlock() // we're done with bot data 'til the end

PlugHandlerLoop:
	for plug, _ := range pluginHandlers {
		if !pNameRe.MatchString(plug) {
			b.Log(Error, "Plugin name \"%s\" doesn't match plugin name regex \"%s\", skipping")
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
			b.Log(Error, "Plugin name duplicates external, skipping:", plug)
		} else {
			pnames[i] = plug
			ptypes[i] = plugGo
			i++
		}
	}
	b.Log(Trace, fmt.Sprintf("pnames: %q", pnames))
	b.Log(Trace, fmt.Sprintf("ptypes: %q", ptypes))
	plist := make([]Plugin, 0, nump)

	// Because some plugins may be disabled, pnames and plugins won't necessarily sync
	plugIndex := 0

PlugLoop:
	for i, plug := range pnames {
		var plugin Plugin
		b.Log(Trace, fmt.Sprintf("Loading plugin #%d - %s, type %d", plugIndex, plug, ptypes[i]))
		// getConfigFile loads stock config, then overlays with local
		err := b.getConfigFile("plugins/"+plug+".json", &plugin)
		if err != nil {
			b.Log(Error, fmt.Errorf("Unable to load configuration for plugin \"%s\": %v", plug, err))
			continue
		}
		if plugin.Disabled {
			continue
		}
		plugin.pluginType = ptypes[i]
		b.Log(Info, "Loaded configuration for plugin", plug)
		// Use bot default plugin channels if none defined, unless AllChannels requested. Admin can override.
		if len(plugin.Channels) == 0 && len(pchan) > 0 && !plugin.AllChannels {
			plugin.Channels = pchan
		}
		b.Log(Trace, fmt.Sprintf("Plugin %s will be active in channels %q", plug, plugin.Channels))
		// Compile the regex's
		for i, _ := range plugin.CommandMatches {
			command := &plugin.CommandMatches[i]
			re, err := regexp.Compile(`^\s*` + command.Regex + `\s*$`)
			if err != nil {
				b.Log(Error, fmt.Errorf("Skipping %s, couldn't compile command regular expression \"%s\": %v", plug, command.Regex, err))
				continue PlugLoop
			}
			command.re = re
		}
		for i, _ := range plugin.ReplyMatchers {
			reply := &plugin.ReplyMatchers[i]
			re, err := regexp.Compile(`^\s*` + reply.Regex + `\s*$`)
			if err != nil {
				b.Log(Error, fmt.Errorf("Skipping %s, couldn't compile reply regular expression \"%s\": %v", plug, reply.Regex, err))
				continue PlugLoop
			}
			reply.re = re
		}
		for i, _ := range plugin.MessageMatches {
			// Note that full message regexes don't get the beginning and end anchors added - the individual plugin
			// will need to do this if necessary.
			message := &plugin.MessageMatches[i]
			re, err := regexp.Compile(message.Regex)
			if err != nil {
				b.Log(Error, fmt.Errorf("Skipping %s, couldn't compile message regular expression \"%s\": %v", plug, message.Regex, err))
				continue PlugLoop
			}
			message.re = re
		}
		plugin.Name = plug
		// Copy the pointer to the empty config struct / empty struct (when no config)
		pt := reflect.ValueOf(pluginHandlers[plug].Config)
		if pt.Kind() == reflect.Ptr {
			// reflect magic: create a pointer to a new empty config struct for the plugin
			plugin.config = reflect.New(reflect.Indirect(pt).Type()).Interface()
			if err := json.Unmarshal(plugin.Config, plugin.config); err != nil {
				b.Log(Error, fmt.Sprintf("Error unmarshalling plugin config json to config: %v", err))
			}
		} else {
			b.Log(Debug, "config interface isn't a pointer, skipping unmarshal for plugin:", plug)
		}
		// Generate the random id
		p := make([]byte, 16)
		random.Read(p)
		plugin.pluginID = fmt.Sprintf("%x", p)
		pfinder[plugin.pluginID] = plugIndex
		// Store this plugin's config in the temporary list
		b.Log(Info, fmt.Sprintf("Recorded plugin #%d, \"%s\" with ID %s", plugIndex, plugin.Name, plugin.pluginID))
		plist = append(plist, plugin)
		plugIndex++
	}

	reInitPlugins := false
	b.lock.Lock()
	b.plugins = plist
	b.plugIDmap = pfinder
	if b.Connector != nil {
		reInitPlugins = true
	}
	b.lock.Unlock()
	if reInitPlugins {
		b.initializePlugins()
	}
}
