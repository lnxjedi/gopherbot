// Package bot provides the interfaces for creating a chatops
// bot.
package bot

/* bot.go defines core data structures and public methods for startup.
   handler.go has the methods for callbacks from the connector, */

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"
)

type VersionInfo struct {
	Version, Commit string
}

// configPath is optional, installPath is where gopherbot(.exe) is
var configPath, installPath string

var botVersion VersionInfo

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
	Connector                             // Connector interface, implemented by each specific protocol
	adminUsers           []string         // List of users with access to administrative commands
	alias                rune             // single-char alias for addressing the bot
	name                 string           // e.g. "Gort"
	fullName             string           // e.g. "Robbie Robot"
	adminContact         string           // who to contact for problems with the robot.
	email                string           // the from: when the robot sends email
	mailConf             botMailer        // configuration to use when sending email
	ignoreUsers          []string         // list of users to never listen to, like other bots
	preRegex             *regexp.Regexp   // regex for matching prefixed commands, e.g. "Gort, drop your weapon"
	postRegex            *regexp.Regexp   // regex for matching, e.g. "open the pod bay doors, hal"
	bareRegex            *regexp.Regexp   // regex for matching the robot's bare name, if you forgot it in the previous command
	joinChannels         []string         // list of channels to join
	defaultAllowDirect   bool             // whether plugins are available in DM by default
	defaultMessageFormat MessageFormat    // Raw unless set to Variable or Fixed
	plugChannels         []string         // list of channels where plugins are available by default
	protocol             string           // Name of the protocol, e.g. "slack"
	brainProvider        string           // Type of Brain provider to use
	brain                SimpleBrain      // Interface for robot to Store and Retrieve data
	brainKey             string           // Configured brain key
	historyProvider      string           // Name of the history provider to use
	history              HistoryProvider  // Provider for storing and retrieving job / plugin histories
	workSpace            string           // Read/Write directory where the robot does work
	defaultElevator      string           // Plugin name for performing elevation
	defaultAuthorizer    string           // Plugin name for performing authorization
	externalPlugins      []externalPlugin // List of external plugins to load
	externalJobs         []externalJob    // List of external jobs to load
	externalTasks        []externalTask   // List of external tasks to load
	scheduledTasks       []scheduledTask  // List of scheduled tasks
	port                 string           // Localhost port to listen on
	stop                 chan struct{}    // stop channel for stopping the connector
	done                 chan struct{}    // channel closed when robot finishes shutting down
	timeZone             *time.Location   // for forcing the TimeZone, Unix only
	defaultJobChannel    string           // where job statuses will post if not otherwise specified
	shuttingDown         bool             // to prevent new plugins from starting
	pluginsRunning       int              // a count of how many plugins are currently running
	paused               bool             // it's a Windows thing
	sync.WaitGroup                        // for keeping track of running plugins
	sync.RWMutex                          // for safe updating of bot data structures
}

var listening bool // for tests where initBot runs multiple times

// initBot sets up the global robot and loads
// configuration.
func initBot(cpath, epath string, logger *log.Logger) {
	stopRegistrations = true
	// Seed the pseudo-random number generator, for plugin IDs, RandomString, etc.
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	botLogger.l = logger

	configPath = cpath
	installPath = epath

	var wd string
	if len(configPath) > 0 {
		wd = configPath
	} else {
		wd = installPath
	}
	err := os.Chdir(wd)
	if err != nil {
		Log(Fatal, fmt.Sprintf("Unable to set initial working directory to '%s': %v", wd, err))
	}
	Log(Info, fmt.Sprintf("Set initial working directory: %s", wd))
	robot.stop = make(chan struct{})
	robot.done = make(chan struct{})
	robot.shuttingDown = false

	handle := handler{}
	bot := &botContext{
		workingDirectory: robot.workSpace,
		environment:      make(map[string]string),
	}
	if err := bot.loadConfig(true); err != nil {
		Log(Fatal, fmt.Sprintf("Error loading initial configuration: %v", err))
	}

	if len(robot.brainProvider) > 0 {
		if bprovider, ok := brains[robot.brainProvider]; !ok {
			Log(Fatal, fmt.Sprintf("No provider registered for brain: \"%s\"", robot.brainProvider))
		} else {
			brain := bprovider(handle, logger)
			robot.brain = brain
		}
	} else {
		bprovider, _ := brains["mem"]
		robot.brain = bprovider(handle, logger)
		Log(Error, "No brain configured, falling back to default 'mem' brain - no memories will persist")
	}
	if encryptBrain {
		if len(robot.brainKey) > 0 {
			if initializeEncryption(robot.brainKey) {
				Log(Info, "Successfully initialized brain encryption")
			} else {
				Log(Error, "Failed to initialize brain encryption with configured BrainKey")
			}
		} else {
			Log(Warn, "Brain encryption specified but no key configured; use 'initialize brain <key>' to initialize the encrypted brain")
		}
	}
	if len(robot.historyProvider) > 0 {
		if hprovider, ok := historyProviders[robot.historyProvider]; !ok {
			Log(Fatal, fmt.Sprintf("No provider registered for history type: \"%s\"", robot.historyProvider))
		} else {
			hp := hprovider(handle)
			robot.history = hp
		}
	}
	if !listening {
		listening = true
		go func() {
			h := handler{}
			http.Handle("/json", h)
			Log(Fatal, http.ListenAndServe(robot.port, nil))
		}()
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
	// Start the brain loop
	go runBrain()

	bot := &botContext{
		workingDirectory: robot.workSpace,
		environment:      make(map[string]string),
	}
	bot.registerActive()
	bot.loadConfig(false)
	bot.deregister()

	var cl []string
	robot.RLock()
	cl = append(cl, robot.joinChannels...)
	robot.RUnlock()
	for _, channel := range cl {
		robot.JoinChannel(channel)
	}

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
	pr := robot.pluginsRunning
	stop := robot.stop
	robot.RUnlock()
	Log(Debug, fmt.Sprintf("stop called with %d plugins running", pr))
	robot.Wait()
	brainQuit()
	close(stop)
}
