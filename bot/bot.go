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
	Connector                       // Connector interface, implemented by each specific protocol
	localPath       string          // Directory for local files overriding default config
	installPath     string          // Path to the bot's installation directory
	level           LogLevel        // Log level for bot methods
	adminUsers      []string        // List of users with access to administrative commands
	alias           rune            // single-char alias for addressing the bot
	name            string          // e.g. "Gort"
	fullName        string          // e.g. "Robbie Robot"
	ignoreUsers     []string        // list of users to never listen to, like other bots
	preRegex        *regexp.Regexp  // regex for matching prefixed commands, e.g. "Gort, drop your weapon"
	postRegex       *regexp.Regexp  // regex for matching, e.g. "open the pod bay doors, hal"
	channels        []string        // list of channels to join
	plugChannels    []string        // list of channels where plugins are active by default
	lock            sync.RWMutex    // for safe updating of bot data structures
	protocol        string          // Name of the protocol, e.g. "slack"
	protocolConfig  json.RawMessage // Raw JSON configuration to pass to the connector
	plugins         []Plugin        // Slice of all the configured plugins
	plugIDmap       map[string]int  // Map of pluginID to it's index in plugins
	externalPlugins []string        // List of external plugins to load
	port            string
}

// gopherBot implements GopherBot for startup
type gopherBot struct {
	bot *robot
	Handler
}

// Create instantiates the one and only instance of a Gobot, and loads
// configuration.
func New(cpath, epath string) (GopherBot, error) {
	botLock.Lock()
	if botCreated {
		botLock.Unlock()
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
	h := handler{bot: b}
	g := gopherBot{bot: b, Handler: h}
	return g, nil
}

// GetConnectorName returns the name of the configured connector
func (g gopherBot) GetConnectorName() string {
	g.bot.lock.RLock()
	proto := g.bot.protocol
	g.bot.lock.RUnlock()
	return proto
}

// Init is called after the bot is connected.
func (g gopherBot) Init(c Connector) {
	b := g.bot
	b.lock.Lock()
	if b.Connector != nil {
		b.lock.Unlock()
		return
	}
	b.Connector = c
	b.lock.Unlock()
	go b.listenHttpJSON()
	var cl []string
	b.lock.RLock()
	cl = append(cl, b.channels...)
	b.lock.RUnlock()
	for _, channel := range cl {
		b.JoinChannel(channel)
	}
	b.initializePlugins()
}
