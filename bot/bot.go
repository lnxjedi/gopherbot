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

// robot holds all the interal data relevant to the Bot. Most of it is populated
// by loadConfig, other stuff is populated by the connector.
type robot struct {
	localPath       string          // Directory for local files overriding default config
	installPath     string          // Path to the bot's installation directory
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
	plugIDmap       map[string]int  // Map of pluginID to it's index in plugins
	externalPlugins []string        // List of external plugins to load
	port            string
}

// Public interface for package main to initialize the robot with a connector
type GopherBot interface {
	GetConnectorName() string
	Init(c Connector)
	Handler // the Connector needs a Handler
}

// Create instantiates the one and only instance of a Gobot, and loads
// configuration.
func Create(cpath, epath string) (GopherBot, error) {
	botLock.Lock()
	if botCreated {
		return nil, fmt.Errorf("bot already created")
	}
	// There can be only one
	botCreated = true
	// Prevent plugin registration after program init
	stopRegistrations = true

	b := &robot{}
	botLock.Unlock()

	b.localPath = cpath
	b.installPath = epath

	if err := b.loadConfig(); err != nil {
		return nil, err
	}
	return GopherBot(b), nil
}

// GetConnectorName returns the name of the configured connector
func (b *robot) GetConnectorName() string {
	return b.protocol
}

// Init is called after the bot is connected.
func (b *robot) Init(c Connector) {
	b.Lock()
	if b.Connector != nil {
		b.Unlock()
		return
	}
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
