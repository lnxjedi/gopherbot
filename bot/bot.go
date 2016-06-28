// Package bot provides the interfaces for creating a chatops
// bot.
package bot

/* bot.go defines core data structures and public methods for startup.
   handler.go has the methods for callbacks from the connector, */

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"sync"
	"time"
)

var botLock sync.RWMutex
var random *rand.Rand

var connectors map[string]func(Handler, *log.Logger) Connector = make(map[string]func(Handler, *log.Logger) Connector)

func RegisterConnector(name string, connstarter func(Handler, *log.Logger) Connector) {
	if stopRegistrations {
		return
	}
	if connectors[name] != nil {
		log.Fatal("Attempted registration of duplicate connector:", name)
	}
	connectors[name] = connstarter
}

// robot holds all the interal data relevant to the Bot. Most of it is populated
// by loadConfig, other stuff is populated by the connector.
type robot struct {
	Connector                        // Connector interface, implemented by each specific protocol
	localPath       string           // Directory for local files overriding default config
	installPath     string           // Path to the bot's installation directory
	level           LogLevel         // Log level for bot methods
	adminUsers      []string         // List of users with access to administrative commands
	alias           rune             // single-char alias for addressing the bot
	name            string           // e.g. "Gort"
	fullName        string           // e.g. "Robbie Robot"
	adminContact    string           // who to contact for problems with the robot.
	email           string           // the from: when the robot sends email
	mailserver      string           // server to use for sending email
	ignoreUsers     []string         // list of users to never listen to, like other bots
	preRegex        *regexp.Regexp   // regex for matching prefixed commands, e.g. "Gort, drop your weapon"
	postRegex       *regexp.Regexp   // regex for matching, e.g. "open the pod bay doors, hal"
	joinChannels    []string         // list of channels to join
	plugChannels    []string         // list of channels where plugins are active by default
	lock            sync.RWMutex     // for safe updating of bot data structures
	protocol        string           // Name of the protocol, e.g. "slack"
	brainProvider   string           // Type of Brain provider to use
	brain           SimpleBrain      // Interface for robot to Store and Retrieve data
	externalPlugins []externalPlugin // List of external plugins to load
	port            string           // Localhost port to listen on
	logger          *log.Logger      // Where to log to
}

var b *robot

// newBot instantiates the one and only instance of a Gobot, and loads
// configuration.
func newBot(cpath, epath string, logger *log.Logger) error {
	botLock.Lock()
	// Prevent plugin registration after program init
	stopRegistrations = true
	// Seed the pseudo-random number generator, for plugin IDs, RandomString, etc.
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	b = &robot{}
	botLock.Unlock()

	b.localPath = cpath
	b.installPath = epath
	b.logger = logger
	b.mailserver = "127.0.0.1:25"

	if err := loadConfig(); err != nil {
		return nil
	}
	if len(b.brainProvider) > 0 {
		provider, ok := brains[b.brainProvider]
		if !ok {
			Log(Fatal, fmt.Sprintf("No provider registered for brain: \"%s\"", b.brainProvider))
		}
		h := handler{}
		b.brain = provider(h, logger)
	}
	return nil
}

// Init is called after the bot is connected.
func botInit(c Connector) {
	b.lock.Lock()
	if b.Connector != nil {
		b.lock.Unlock()
		return
	}
	b.Connector = c
	b.lock.Unlock()
	go listenHttpJSON()
	var cl []string
	b.lock.RLock()
	cl = append(cl, b.joinChannels...)
	b.lock.RUnlock()
	for _, channel := range cl {
		b.JoinChannel(channel)
	}
	initializePlugins()
}
