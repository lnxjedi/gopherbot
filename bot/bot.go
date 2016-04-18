// Package bot provides the interfaces for creating a chatops
// bot.
package bot

/* bot.go defines core data structures and public methods for startup
   and the connector, but not exposed to plugins. */

import (
	"encoding/json"
	"log"
	"regexp"
	"sync"
)

var botLock sync.RWMutex
var botCreated bool

// Bot holds all the interal data relevant to the Bot. Most of it is populated
// by LoadConfig, other stuff is populated by the connector.
type Bot struct {
	configPath      string          // directory holding the bot's json config files
	level           LogLevel        // Log level for bot methods
	alias           rune            // single-char alias for addressing the bot
	name            string          // e.g. "Gort"
	ignoreUsers     []string        // list of users to never listen to, like other bots
	preRegex        *regexp.Regexp  // regex for matching prefixed commands, e.g. "Gort, drop your weapon"
	postRegex       *regexp.Regexp  // regex for matching, e.g. "open the pod bay doors, hal"
	channels        []string        // list of channels to join
	plugChannels    []string        // list of channels where plugins are active by default
	sync.RWMutex                    // for safe updating of bot data structures
	Connector                       // Connector interface, implemented by each specific protocol
	protocolConfig  json.RawMessage // Raw JSON configuration to pass to the connector
	plugins         []Plugin        // Slice of all the configured plugins
	externalPlugins []string        // List of external plugins to load
	port            string
}

// LogLevel for determining when to output a log entry
type LogLevel int

// Definitions of log levels in order from most to least verbose
const (
	Trace LogLevel = iota
	Debug
	Info
	Warn
	Error
)

// Instantiate the one and only instance of a Gobot
func Create() *Bot {
	botLock.Lock()
	if botCreated {
		return nil
	}
	// There can be only one
	botCreated = true
	// Prevent plugin registration after program init
	stopRegistrations = true
	b := &Bot{}
	botLock.Unlock()
	return b
}

// GetProtocolConfig returns the connector protocol's json.RawMessage to the connector
func (b *Bot) GetProtocolConfig() json.RawMessage {
	var pc []byte
	b.RLock()
	// Make of copy of the protocol config for the plugin
	pc = append(pc, []byte(b.protocolConfig)...)
	b.RUnlock()
	return pc
}

// Log logs messages whenever the connector log level is
// less than the given level
func (b *Bot) Log(l LogLevel, v ...interface{}) {
	if l >= b.level {
		var prefix string
		switch l {
		case Trace:
			prefix = "Trace:"
		case Debug:
			prefix = "Debug:"
		case Info:
			prefix = "Info"
		case Warn:
			prefix = "Warning:"
		case Error:
			prefix = "Error"
		}
		log.Println(prefix, v)
	}
}

// SetLogLevel updates the connector log level
func (b *Bot) SetLogLevel(l LogLevel) {
	b.Lock()
	b.level = l
	b.Unlock()
}

// GetLogLevel returns the current log level
func (b *Bot) GetLogLevel() LogLevel {
	b.RLock()
	l := b.level
	b.RUnlock()
	return l
}

// Connectors that support it can call SetName; otherwise it should
// be configured in gobot.conf.
func (b *Bot) SetName(n string) {
	b.Lock()
	b.Log(Debug, "Setting name to: "+n)
	b.name = n
	b.Unlock()
	b.updateRegexes()
}

// Init is called after the bot is connected.
func (b *Bot) Init(c Connector) {
	b.Lock()
	b.Connector = c
	b.Unlock()
	go b.listenHttpJSON()
	var cl []string
	b.RLock()
	cl = append(cl, b.channels...)
	b.RUnlock()
	for _, channel := range cl {
		b.JoinChannel(channel)
	}
	b.initializePlugins()
	//TODO: remove this later
	name, _ := b.GetProtocolUserAttribute("davidp", "realName")
	b.SendUserMessage("davidp", "Hello, sir! I know who you are now: "+name)
	//	b.conn.SendChannelMessage("C0RK4DG68", "Hello, World!")
}
