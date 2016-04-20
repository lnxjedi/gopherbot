// Package bot provides the interfaces for creating a chatops
// bot.
package bot

/* bot.go defines core data structures and public methods for startup.
   handler.go has the methods for callbacks from the connector, */

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
)

var botLock sync.RWMutex
var botCreated bool

// Bot holds all the interal data relevant to the Bot. Most of it is populated
// by LoadConfig, other stuff is populated by the connector.
type Bot struct {
	configPath      string          // directory holding the bot's json config files
	execPath        string          // Path to the bot's installation directory
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
	protocol        string          // Name of the protocol, e.g. "slack"
	protocolConfig  json.RawMessage // Raw JSON configuration to pass to the connector
	plugins         []Plugin        // Slice of all the configured plugins
	externalPlugins []string        // List of external plugins to load
	port            string
}

// Create instantiates the one and only instance of a Gobot, and loads
// configuration.
func Create(cpath, epath string) (*Bot, error) {
	botLock.Lock()
	if botCreated {
		return nil, fmt.Errorf("bot already created")
	}
	// There can be only one
	botCreated = true
	// Prevent plugin registration after program init
	stopRegistrations = true

	b := &Bot{}
	botLock.Unlock()

	b.configPath = cpath
	b.execPath = epath

	if err := b.LoadConfig(); err != nil {
		return nil, err
	}
	return b, nil
}

// GetConnectorName returns the name of the configured connector
func (b *Bot) GetConnectorName() string {
	return b.protocol
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
}
