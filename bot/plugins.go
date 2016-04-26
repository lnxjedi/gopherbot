package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"time"
)

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

// Plugin specifies the structure of a plugin configuration - plugins should include an example
type Plugin struct {
	Name           string          // the name of the plugin, used as a key in to the
	pluginType     plugType        // plugGo, plugExternal, plugBuiltin - determines how commands are routed
	PluginPath     string          // Path to the external executable that expects <channel> <user> <command> <arg> <arg> from regex matches - for Plugtype=shell only
	Disabled       bool            // Set true to disable the plugin
	DisallowDirect bool            // Set this true if this plugin can never be accessed via direct message
	Channels       []string        // Channels where the plugin is active - rifraf like "memes" should probably only be in random, but it's configurable. If empty uses DefaultChannels
	Help           []PluginHelp    // All the keyword sets / help texts for this plugin
	CommandMatches []InputMatcher  // Input matchers for messages that need to be directed to the 'bot
	MessageMatches []InputMatcher  // Input matchers for messages the 'bot hears even when it's not being spoken to
	Config         json.RawMessage // Plugin Configuration - the plugin needs to decode this
	pluginID       string          // 32-char random ID for identifying plugins in callbacks
}

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
		if handler, ok := pluginHandlers[plugin.Name]; ok {
			bot.pluginID = plugin.pluginID
			go handler(bot, "", b.name, "init")
		}
	}
}

// pluginHandlers maps from plugin names to handler functions; populated during package initialization and never written to again.
var pluginHandlers map[string]func(bot Robot, command string, args ...string) = make(map[string]func(bot Robot, command string, args ...string))

// stopRegistrations is set "true" when the bot is created to prevent registration outside of init functions
var stopRegistrations bool = false

// RegisterPlugin allows plugins to register a handler function in a func init().
// When the bot initializes, it will call each plugin's handler with a command
// "init", empty channel, the bot's username, and no arguments, so the plugin
// can store this information for, e.g., scheduled jobs.
func RegisterPlugin(name string, handler func(bot Robot, command string, args ...string)) {
	if stopRegistrations {
		return
	}
	if pluginHandlers[name] != nil {
		log.Fatal("Attempted registration of duplicate plugin name:", name)
	}
	pluginHandlers[name] = handler
}

// loadPluginConfig() loads the configuration for all the plugins from
// $GOPHER_LOCALDIR/plugins/<pluginname>.json (excepting builtins), assigns
// a pluginID, and stores the resulting array in b.plugins. Bad plugins
// are skipped and logged.
func (b *robot) loadPluginConfig() {
	i := 0

	// Seed the pseudo-random number generator, for plugin IDs
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
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
		pnames[i] = plugin.Name
		pset[plugin.Name] = true
		ptypes[i] = plugBuiltin
		i++
	}

	for _, plug := range b.externalPlugins {
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
	pchan := make([]string, 0, len(b.channels))
	pchan = append(pchan, b.channels...)
	b.lock.RUnlock() // we're done with bot data 'til the end

PlugHandlerLoop:
	for plug, _ := range pluginHandlers {
		if pset[plug] { // have to check builtIns, already loaded
			for _, plugin := range builtIns {
				if plug == plugin.Name {
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
	plist := make([]Plugin, nump)

	// Because some plugins may be disabled, pnames and plugins won't necessarily sync
	plugIndex := 0

PlugLoop:
	for i, plug := range pnames {
		var plugin Plugin
		b.Log(Trace, fmt.Sprintf("Loading plugin #%d - %s, type %d", i, plug, ptypes[i]))
		if ptypes[i] == plugBuiltin {
			plugin = builtIns[i]
		} else {
			// getConfigFile loads stock config, then overlays with local
			err := b.getConfigFile("plugins/"+plug+".json", &plugin)
			if err != nil {
				b.Log(Error, fmt.Errorf("Unable to load configuration for plugin \"%s\": %v", plug, err))
				continue
			}
		}
		if plugin.Disabled {
			continue
		}
		plugin.pluginType = ptypes[i]
		b.Log(Info, "Loaded configuration for plugin", plug)
		// Use bot default plugin channels if none defined; only builtins default to all channels
		if len(plugin.Channels) == 0 && len(pchan) > 0 && plugin.pluginType != plugBuiltin {
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
		// Generate the random id
		p := make([]byte, 16)
		_, rerr := r.Read(p)
		if rerr != nil {
			log.Fatal("Couldn't generate plugin id:", rerr)
		}
		plugin.pluginID = fmt.Sprintf("%x", p)
		pfinder[plugin.pluginID] = plugIndex
		// Store this plugin's config in the temporary list
		b.Log(Info, fmt.Sprintf("Recorded plugin #%d, \"%s\" with ID %s", plugIndex, plugin.Name, plugin.pluginID))
		plist[plugIndex] = plugin
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

// handleMessage checks the message against plugin commands and full-message matches,
// then dispatches it to all applicable handlers in a separate go routine.

func (b *robot) handleMessage(isCommand bool, channel, user, messagetext string) {
	b.lock.RLock()
	bot := Robot{
		User:    user,
		Channel: channel,
		Format:  Variable,
		robot:   b,
	}
	if len(channel) == 0 {
		b.Log(Trace, fmt.Sprintf("Bot received a direct message from %s: %s", user, messagetext))
	}
	for _, plugin := range b.plugins {
		b.Log(Trace, fmt.Sprintf("Checking message \"%s\" against plugin %s, active in %d channels", messagetext, plugin.Name, len(plugin.Channels)))
		ok := b.messageAppliesToPlugin(user, channel, messagetext, plugin)
		if !ok {
			b.Log(Trace, fmt.Sprintf("Plugin %s ignoring message in channel %s, doesn't meet criteria", plugin.Name, channel))
			continue
		}
		var matchers []InputMatcher
		if isCommand {
			matchers = plugin.CommandMatches
		} else {
			matchers = plugin.MessageMatches
		}
		for _, matcher := range matchers {
			b.Log(Trace, fmt.Sprintf("Checking \"%s\" against \"%s\"", messagetext, matcher.Regex))
			matches := matcher.re.FindAllStringSubmatch(messagetext, -1)
			if matches != nil {
				b.Log(Debug, fmt.Sprintf("Dispatching command %s to plugin %s", matcher.Command, plugin.Name))
				bot.pluginID = plugin.pluginID
				switch plugin.pluginType {
				case plugBuiltin, plugGo:
					go pluginHandlers[plugin.Name](bot, matcher.Command, matches[0][1:]...)
				case plugExternal:
					var fullPath string // full path to the executable
					if len(plugin.PluginPath) == 0 {
						b.Log(Error, "PluginPath empty for external plugin:", plugin.Name)
					}
					if byte(plugin.PluginPath[0]) == byte("/"[0]) {
						fullPath = plugin.PluginPath
					} else {
						_, err := os.Stat(b.localPath + "/" + plugin.PluginPath)
						if err != nil {
							_, err := os.Stat(b.installPath + "/" + plugin.PluginPath)
							if err != nil {
								b.Log(Error, fmt.Errorf("Couldn't locate external plugin %s: %v", plugin.Name, err))
								continue
							}
							fullPath = b.installPath + "/" + plugin.PluginPath
							b.Log(Debug, "Using stock external plugin:", fullPath)
						} else {
							fullPath = b.localPath + "/" + plugin.PluginPath
							b.Log(Debug, "Using local external plugin:", fullPath)
						}
					}
					args := make([]string, 0, 3+len(matches[0])-1)
					args = append(args, channel, user, matcher.Command, plugin.pluginID)
					args = append(args, matches[0][1:]...)
					b.Log(Trace, fmt.Sprintf("Calling \"%s\" with args: %q", fullPath, args))
					// cmd := exec.Command(fullPath, channel, user, matcher.Command, matches[0][1:]...)
					cmd := exec.Command(fullPath, args...)
					cmd.Stdout = nil
					stderr, err := cmd.StderrPipe()
					if err != nil {
						b.Log(Error, fmt.Errorf("Creating stderr pipe for external command \"%s\": %v", fullPath, err))
						continue
					}
					go func() {
						if err := cmd.Start(); err != nil {
							b.Log(Error, fmt.Errorf("Starting command \"%s\": %v", fullPath, err))
							return
						}
						defer func() {
							if err := cmd.Wait(); err != nil {
								b.Log(Error, fmt.Errorf("Waiting on external command \"%s\": %v", fullPath, err))
							}
						}()
						stdErrBytes, err := ioutil.ReadAll(stderr)
						if err != nil {
							b.Log(Error, fmt.Errorf("Reading from stderr for external command \"%s\": %v", fullPath, err))
							return
						}
						stdErrString := string(stdErrBytes)
						if len(stdErrString) > 0 {
							b.Log(Warn, fmt.Errorf("Output from stderr of external command \"%s\": %s", fullPath, stdErrString))
						}
					}()
				}
			}
		}
	}
	b.lock.RUnlock()
}
