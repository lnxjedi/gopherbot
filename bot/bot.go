// Package bot provides the interfaces for creating a chatops
// bot.
package bot

/* bot.go defines core data structures and public methods for startup.
   handler.go has the methods for callbacks from the connector, */

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
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
	bareRegex          *regexp.Regexp   // regex for matching the robot's bare name, if you forgot it in the previous command
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
	stop               chan struct{}    // stop channel for stopping the connector
	done               chan struct{}    // channel closed when robot finishes shutting down
	events             chan Event       // buffered channel for message disposition events used by integration testing
	shuttingDown       bool             // to prevent new plugins from starting
	pluginsRunning     int              // a count of how many plugins are currently running
	paused             bool             // it's a Windows thing
	sync.WaitGroup                      // for keeping track of running plugins
}

// initBot sets up the global robot and loads
// configuration.
func initBot(cpath, epath string, logger *log.Logger) {
	globalLock.Lock()
	// Prevent plugin registration after program init
	stopRegistrations = true
	// Seed the pseudo-random number generator, for plugin IDs, RandomString, etc.
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	globalLock.Unlock()

	robot.Lock()
	robot.localPath = cpath
	robot.installPath = epath
	robot.logger = logger
	robot.stop = make(chan struct{})
	robot.done = make(chan struct{})
	robot.events = make(chan Event, 16)
	robot.shuttingDown = false
	robot.Unlock()

	handle := handler{}
	if err := loadConfig(); err != nil {
		Log(Fatal, fmt.Sprintf("Error loading initial configuration: %v", err))
	}

	if len(robot.brainProvider) > 0 {
		if bprovider, ok := brains[robot.brainProvider]; !ok {
			Log(Fatal, fmt.Sprintf("No provider registered for brain: \"%s\"", robot.brainProvider))
		} else {
			brain := bprovider(handle, logger)
			robot.Lock()
			robot.brain = brain
			robot.Unlock()
		}
	}
}

// set connector sets the connector, which should already be initialized
func setConnector(c Connector) {
	robot.Lock()
	robot.Connector = c
	robot.Unlock()
}

// run starts all the loops and returns a channel that closes when the robot
// shuts down. It should return after the connector loop has started and
// plugins are initialized.
func run() <-chan struct{} {
	robot.RLock()
	port := robot.port
	robot.RUnlock()
	if len(port) > 0 {
		// Only start the HttpListener once, runs for life of process
		botHttpListener.Lock()
		if !botHttpListener.listening {
			botHttpListener.listening = true
			go listenHTTPJSON()
		}
		botHttpListener.Unlock()
	}

	var cl []string
	robot.RLock()
	cl = append(cl, robot.joinChannels...)
	robot.RUnlock()
	for _, channel := range cl {
		robot.JoinChannel(channel)
	}

	// Start the brain loop
	go runBrain()

	// signal handler
	go func() {
		robot.RLock()
		done := robot.done
		robot.RUnlock()
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	loop:
		for {
			select {
			case sig := <-sigs:
				robot.Lock()
				if robot.shuttingDown {
					Log(Warn, "Received SIGINT/SIGTERM while shutdown in progress")
					robot.Unlock()
				} else {
					robot.shuttingDown = true
					robot.Unlock()
					signal.Stop(sigs)
					Log(Info, fmt.Sprintf("Exiting on signal: %s", sig))
					stop()
				}
			case <-done:
				break loop
			}
		}
	}()

	// connector loop
	robot.RLock()
	go func(conn Connector, stop <-chan struct{}, done chan<- struct{}) {
		conn.Run(stop)
		close(done)
	}(robot.Connector, robot.stop, robot.done)
	robot.RUnlock()

	initializePlugins()
	robot.RLock()
	defer robot.RUnlock()
	return robot.done
}

// stop is called whenever the robot needs to shut down gracefully. All callers
// should lock the bot and check the value of robot.shuttingDown; see
// builtins.go and win_svc_run.go
func stop() {
	robot.RLock()
	Log(Debug, fmt.Sprintf("stop called with %d plugins running", robot.pluginsRunning))
	stop := robot.stop
	robot.RUnlock()
	robot.Wait()
	brainQuit()
	close(stop)
}
