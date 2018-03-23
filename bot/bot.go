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

// Version is the current version of Gopherbot
var Version = "v1.1.0-snapshot"

// mkdist.* creates a temporary commit.go that sets commit to the current
// git commit in an init() function
var commit = "(manual build)"

var globalLock sync.RWMutex
var random *rand.Rand

var connectors = make(map[string]func(Handler, *log.Logger) Connector)

// RegisterConnector should be called in an init function to register a type
// of connector. Currently only Slack is implemented.
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
var robot struct {
	Connector                           // Connector interface, implemented by each specific protocol
	localPath          string           // Directory for local files overriding default config
	installPath        string           // Path to the bot's installation directory
	adminUsers         []string         // List of users with access to administrative commands
	alias              rune             // single-char alias for addressing the bot
	name               string           // e.g. "Gort"
	fullName           string           // e.g. "Robbie Robot"
	adminContact       string           // who to contact for problems with the robot.
	email              string           // the from: when the robot sends email
	mailConf           botMailer        // configuration to use when sending email
	ignoreUsers        []string         // list of users to never listen to, like other bots
	preRegex           *regexp.Regexp   // regex for matching prefixed commands, e.g. "Gort, drop your weapon"
	postRegex          *regexp.Regexp   // regex for matching, e.g. "open the pod bay doors, hal"
	joinChannels       []string         // list of channels to join
	defaultAllowDirect bool             // whether plugins are available in DM by default
	plugChannels       []string         // list of channels where plugins are active by default
	sync.RWMutex                        // for safe updating of bot data structures
	protocol           string           // Name of the protocol, e.g. "slack"
	brainProvider      string           // Type of Brain provider to use
	brain              SimpleBrain      // Interface for robot to Store and Retrieve data
	defaultElevator    string           // Plugin name for performing elevation
	defaultAuthorizer  string           // Plugin name for performing authorization
	externalPlugins    []externalPlugin // List of external plugins to load
	port               string           // Localhost port to listen on
	logger             *log.Logger      // Where to log to
}

//var robot *robotcfg

// newBot instantiates the one and only instance of a robot, and loads
// configuration.
func newBot(cpath, epath string, logger *log.Logger) error {
	globalLock.Lock()
	// Prevent plugin registration after program init
	stopRegistrations = true
	// Seed the pseudo-random number generator, for plugin IDs, RandomString, etc.
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	globalLock.Unlock()

	robot.localPath = cpath
	robot.installPath = epath
	robot.logger = logger

	handle := handler{}
	if err := loadConfig(); err != nil {
		return err
	}

	if len(robot.brainProvider) > 0 {
		if bprovider, ok := brains[robot.brainProvider]; !ok {
			Log(Fatal, fmt.Sprintf("No provider registered for brain: \"%s\"", robot.brainProvider))
		} else {
			robot.brain = bprovider(handle, logger)
		}
	}
	return nil
}

// Init is called after the bot is connected.
func botInit(c Connector) {
	robot.Lock()
	if robot.Connector != nil {
		robot.Unlock()
		return
	}
	robot.Connector = c
	robot.Unlock()
	go listenHTTPJSON()
	var cl []string
	robot.RLock()
	cl = append(cl, robot.joinChannels...)
	robot.RUnlock()
	for _, channel := range cl {
		robot.JoinChannel(channel)
	}
	initializePlugins()
}
