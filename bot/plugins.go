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

// interface Gobot defines the API for plugins
type Gobot interface {
	Chatbot
	BotLogger
}

// Robot is passed to the plugin to enable convenience functions Say and Reply
type Robot struct {
	User     string // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel  string // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Format   string // The outgoing message format, one of "fixed", "variable"
	pluginID string // Pass the ID in for later identificaton of the plugin
	Gobot
}

// TODO: implement Say and Reply convenience functions

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

// Plugin specifies the structure of a plugin configuration - plugins should include an example
type Plugin struct {
	Name           string          // the name of the plugin
	PluginType     string          // "go" or "external", determines how commands are interpreted
	PluginPath     string          // Path to the external executable that expects <channel> <user> <command> <arg> <arg> from regex matches - for Plugtype=shell only
	AllowDirect    bool            // Whether or not the plugin responds to direct messages
	Channels       []string        // Channels where the plugin is active - rifraf like "memes" should probably only be in random, but it's configurable. If empty uses DefaultChannels
	Help           []PluginHelp    // All the keyword sets / help texts for this plugin
	CommandMatches []InputMatcher  // Input matchers for messages that need to be directed to the 'bot
	MessageMatches []InputMatcher  // Input matchers for messages the 'bot hears even when it's not being spoken to
	Config         json.RawMessage // Plugin Configuration - the plugin needs to decode this
	pluginID       string          // 32-char random ID for identifying plugins in callbacks
}

// initialize sends the "start" command to every plugin
func (b *Bot) initializePlugins() {
	bot := Robot{
		User:    b.name,
		Channel: "",
		Format:  "variable",
		Gobot:   b,
	}
	for _, handler := range goPluginHandlers {
		go handler(bot, "", b.name, "start")
	}
}

// goPluginHandlers maps from plugin names to handler functions; populated during package initialization and never written to again.
var goPluginHandlers map[string]func(bot Robot, channel, user, command string, args ...string) error = make(map[string]func(bot Robot, channel, user, command string, args ...string) error)

// stopRegistrations is set "true" when the bot is created to prevent registration outside of init functions
var stopRegistrations bool = false

// RegisterPlugin allows plugins to register a handler function in a func init().
// When the bot initializes, it will call each plugin's handler with a command
// "start", empty channel, the bot's username, and no arguments, so the plugin
// can store this information for, e.g., scheduled jobs.
func RegisterPlugin(name string, handler func(bot Robot, channel, user, command string, args ...string) error) {
	if stopRegistrations {
		return
	}
	goPluginHandlers[name] = handler
}

// handle checks the message against plugin commands and full-message matches,
// then dispatches it to all applicable handlers.
func (b *Bot) handleMessage(isCommand bool, channel, user, messagetext string) {
	b.RLock()
	bot := Robot{
		User:    user,
		Channel: channel,
		Format:  "variable",
		Gobot:   b,
	}
	for _, plugin := range b.plugins {
		if len(plugin.Channels) > 0 {
			ok := false
			if len(channel) > 0 {
				for _, pchannel := range plugin.Channels {
					if pchannel == channel {
						ok = true
					}
				}
			} else {
				b.Log(Debug, fmt.Sprintf("Checking whether direct messages allowed for %s, AllowDirect is %b", plugin.Name, plugin.AllowDirect))
				if plugin.AllowDirect {
					ok = true
				}
			}
			if !ok {
				b.Log(Trace, fmt.Sprintf("Plugin %s ignoring message in channel %s, not in list", plugin.Name, channel))
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
					switch plugin.PluginType {
					case "go":
						go goPluginHandlers[plugin.Name](bot, channel, user, matcher.Command, matches[0][1:]...)
						//case "external":
					case "external":
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
						args = append(args, channel, user, matcher.Command)
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
					default:
						b.Log(Error, fmt.Sprintf("Invalid plugin type \"%s\" for plugin \"%s\"", plugin.PluginType, plugin.Name))
					}
				}
			}
		}
	}
	b.RUnlock()
}

// loadPluginConfig() loads the configuration for all the plugins from
// $GOPHER_LOCALDIR/plugins/<pluginname>.json
func (b *Bot) loadPluginConfig() error {
	i := 0

	// Seed the pseudo-random number generator, for plugin IDs
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Copy some data from the bot under lock
	b.RLock()
	// Get a list of all plugins from the package goPluginHandlers var and
	// the list of external plugins
	nump := len(goPluginHandlers) + len(b.externalPlugins)
	pnames := make([]string, nump)

	for _, plug := range b.externalPlugins {
		pnames[i] = plug
		i++
	}
	pchan := make([]string, 0, len(b.channels))
	pchan = append(pchan, b.channels...)
	b.RUnlock()

	for plug, _ := range goPluginHandlers {
		pnames[i] = plug
		i++
	}
	plist := make([]Plugin, nump)

	i = 0
	for _, plug := range pnames {
		pc, err := b.getConfigFile("plugins/" + plug + ".json")
		if err != nil {
			return fmt.Errorf("Loading configuration for plugin %s: %v", plug, err)
		}
		var plugin Plugin
		if err := json.Unmarshal(pc, &plugin); err != nil {
			return fmt.Errorf("Unmarshalling JSON for plugin %s: %v", plug, err)
		}
		b.Log(Info, "Loaded configuration for plugin", plug)
		// Use bot default plugin channels if none defined
		if len(plugin.Channels) == 0 && len(pchan) > 0 {
			plugin.Channels = pchan
		}
		b.Log(Trace, fmt.Sprintf("Plugin %s will be active in channels %q", plug, plugin.Channels))
		// Compile the regex's
		for i, _ := range plugin.CommandMatches {
			command := &plugin.CommandMatches[i]
			re, err := regexp.Compile(`^\s*` + command.Regex + `\s*$`)
			if err != nil {
				return fmt.Errorf("Compiling command regular expression %s for plugin %s: %v", command.Regex, plug, err)
			}
			command.re = re
		}
		for i, _ := range plugin.MessageMatches {
			// Note that full message regexes don't get the beginning and end anchors added
			message := &plugin.CommandMatches[i]
			re, err := regexp.Compile(message.Regex)
			if err != nil {
				return fmt.Errorf("Compiling message regular expression %s for plugin %s: %v", message.Regex, plug, err)
			}
			message.re = re
		}
		plugin.Name = plug
		// Generate the random id
		p := make([]byte, 16)
		_, rerr := r.Read(p)
		if rerr != nil {
			log.Fatal("Couldn't generate plugin id:", err)
		}
		plugin.pluginID = fmt.Sprintf("%x", p)
		// Store this plugin's config in the temporary list
		b.Log(Info, fmt.Sprintf("Recorded plugin %s with ID %s", plugin.Name, plugin.pluginID))
		plist[i] = plugin
		i++
	}

	b.Lock()
	b.plugins = plist
	b.Unlock()

	return nil
}
