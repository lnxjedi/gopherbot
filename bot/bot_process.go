// Package bot provides the internal machinery for most of Gopherbot.
package bot

/* bot.go defines core data structures and public methods for startup.
   handler.go has the methods for callbacks from the connector, */

import (
	"encoding/base64"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// VersionInfo holds information about the version, duh. (stupid linter)
type VersionInfo struct {
	Version, Commit string
}

// global values for GOPHER_HOME, GOPHER_CONFIGDIR and GOPHER_INSTALLDIR
var homePath, configPath, installPath string

var botVersion VersionInfo

var random *rand.Rand

var connectors = make(map[string]func(robot.Handler, *log.Logger) robot.Connector)

// RegisterConnector should be called in an init function to register a type
// of connector. Currently only Slack is implemented.
func RegisterConnector(name string, connstarter func(robot.Handler, *log.Logger) robot.Connector) {
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
var botCfg struct {
	robot.Connector                            // Connector interface, implemented by each specific protocol
	adminUsers           []string              // List of users with access to administrative commands
	alias                rune                  // single-char alias for addressing the bot
	botinfo              UserInfo              // robot's name, ID, email, etc.
	adminContact         string                // who to contact for problems with the bot
	mailConf             botMailer             // configuration to use when sending email
	ignoreUsers          []string              // list of users to never listen to, like other bots
	preRegex             *regexp.Regexp        // regex for matching prefixed commands, e.g. "Gort, drop your weapon"
	postRegex            *regexp.Regexp        // regex for matching, e.g. "open the pod bay doors, hal"
	bareRegex            *regexp.Regexp        // regex for matching the robot's bare name, if you forgot it in the previous command
	joinChannels         []string              // list of channels to join
	defaultAllowDirect   bool                  // whether plugins are available in DM by default
	defaultMessageFormat robot.MessageFormat   // Raw unless set to Variable or Fixed
	plugChannels         []string              // list of channels where plugins are available by default
	protocol             string                // Name of the protocol, e.g. "slack"
	brainProvider        string                // Type of Brain provider to use
	brain                robot.SimpleBrain     // Interface for robot to Store and Retrieve data
	encryptionKey        string                // Key for encrypting data (unlocks "real" key in brain)
	historyProvider      string                // Name of the history provider to use
	history              robot.HistoryProvider // Provider for storing and retrieving job / plugin histories
	workSpace            string                // Read/Write directory where the robot does work
	defaultElevator      string                // Plugin name for performing elevation
	defaultAuthorizer    string                // Plugin name for performing authorization
	externalPlugins      []ExternalTask        // List of external plugins to load
	externalJobs         []ExternalTask        // List of external jobs to load
	externalTasks        []ExternalTask        // List of external tasks to load
	loadableModules      []LoadableModule      // List of loadable modules to load
	ScheduledJobs        []ScheduledTask       // List of scheduled tasks
	port                 string                // Localhost port to listen on
	stop                 chan struct{}         // stop channel for stopping the connector
	done                 chan bool             // shutdown channel, true to restart
	timeZone             *time.Location        // for forcing the TimeZone, Unix only
	defaultJobChannel    string                // where job statuses will post if not otherwise specified
	shuttingDown         bool                  // to prevent new plugins from starting
	restart              bool                  // indicate stop and restart vs. stop only, for bootstrapping
	pluginsRunning       int                   // a count of how many plugins are currently running
	sync.WaitGroup                             // for keeping track of running plugins
	sync.RWMutex                               // for safe updating of bot data structures
}

var listening bool // for tests where initBot runs multiple times

// initBot sets up the global robot and loads
// configuration.
func initBot(hpath, cpath, epath string, logger *log.Logger) {
	// Seed the pseudo-random number generator, for plugin IDs, RandomString, etc.
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	botLogger.l = logger

	homePath = hpath
	configPath = cpath
	installPath = epath
	botCfg.stop = make(chan struct{})
	botCfg.done = make(chan bool)
	botCfg.shuttingDown = false

	// Initialize encryption (new style for v2)
	keyEnv := "GOPHER_ENCRYPTION_KEY"
	keyFile := filepath.Join(configPath, "binary-encrypted-key")
	encryptionInitialized := false
	if ek, ok := os.LookupEnv(keyEnv); ok {
		ik := []byte(ek)[0:32]
		if bkf, err := ioutil.ReadFile(keyFile); err == nil {
			if bke, err := base64.StdEncoding.DecodeString(string(bkf)); err == nil {
				if key, err := decrypt(bke, ik); err == nil {
					cryptKey.key = key
					cryptKey.initialized = true
					encryptionInitialized = true
					Log(robot.Info, "Successfully decrypted binary encryption key '%s'", keyFile)
				} else {
					Log(robot.Error, "Decrypting binary encryption key '%s' from environment key '%s': %v", keyFile, keyEnv, err)
				}
			} else {
				Log(robot.Error, "Base64 decoding '%s': %v", keyFile, err)
			}
		} else {
			Log(robot.Warn, "Binary encryption key not loaded from '%s': %v", keyFile, err)
		}
	} else {
		Log(robot.Warn, "GOPHER_ENCRYPTION_KEY not set in environment")
	}

	c := &botContext{
		environment: make(map[string]string),
	}
	if err := c.loadConfig(true); err != nil {
		Log(robot.Fatal, "Loading initial configuration: %v", err)
	}
	os.Unsetenv(keyEnv)

	// loadModules for go loadable modules; a no-op for static builds
	loadModules()

	// All pluggables registered, ok to stop registrations
	stopRegistrations = true

	if len(botCfg.brainProvider) > 0 {
		if bprovider, ok := brains[botCfg.brainProvider]; !ok {
			Log(robot.Fatal, "No provider registered for brain: \"%s\"", botCfg.brainProvider)
		} else {
			brain := bprovider(handle)
			botCfg.brain = brain
			Log(robot.Info, "Initialized brain provider '%s'", botCfg.brainProvider)
		}
	} else {
		bprovider, _ := brains["mem"]
		botCfg.brain = bprovider(handle)
		Log(robot.Error, "No brain configured, falling back to default 'mem' brain - no memories will persist")
	}
	if !encryptionInitialized && len(botCfg.encryptionKey) > 0 {
		if initializeEncryptionFromBrain(botCfg.encryptionKey) {
			Log(robot.Info, "Successfully initialized encryption from configured key")
			encryptionInitialized = true
		} else {
			Log(robot.Error, "Failed to initialize brain encryption with configured EncryptionKey")
		}
	}
	if encryptBrain && !encryptionInitialized {
		Log(robot.Warn, "Brain encryption specified but not initialized; use 'initialize brain <key>' to initialize the encrypted brain interactively")
	}
	if !listening {
		listening = true
		go func() {
			raiseThreadPriv("http handler")
			http.Handle("/json", handle)
			Log(robot.Fatal, "error serving '/json': %s", http.ListenAndServe(botCfg.port, nil))
		}()
	}
}

// set connector sets the connector, which should already be initialized
func setConnector(c robot.Connector) {
	botCfg.Lock()
	botCfg.Connector = c
	botCfg.Unlock()
}

// run starts all the loops and returns a channel that closes when the robot
// shuts down. It should return after the connector loop has started and
// plugins are initialized.
func run() <-chan bool {
	// Start the brain loop
	go runBrain()

	c := &botContext{
		environment: make(map[string]string),
	}
	c.registerActive(nil)
	c.loadConfig(false)
	c.deregister()

	var cl []string
	botCfg.RLock()
	cl = append(cl, botCfg.joinChannels...)
	cl = append(cl, botCfg.plugChannels...)
	cl = append(cl, botCfg.defaultJobChannel)
	botCfg.RUnlock()
	jc := make(map[string]bool)
	for _, channel := range cl {
		if _, ok := jc[channel]; !ok {
			jc[channel] = true
			botCfg.JoinChannel(channel)
		}
	}

	// signal handler
	go func() {
		botCfg.RLock()
		done := botCfg.done
		botCfg.RUnlock()
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	loop:
		for {
			select {
			case sig := <-sigs:
				botCfg.Lock()
				if botCfg.shuttingDown {
					Log(robot.Warn, "Received SIGINT/SIGTERM while shutdown in progress")
					botCfg.Unlock()
				} else {
					botCfg.shuttingDown = true
					botCfg.Unlock()
					signal.Stop(sigs)
					Log(robot.Info, "Exiting on signal: %s", sig)
					stop()
				}
			case <-done:
				break loop
			}
		}
	}()

	// connector loop
	botCfg.RLock()
	go func(conn robot.Connector, stop <-chan struct{}, done chan<- bool) {
		raiseThreadPriv("connector loop")
		conn.Run(stop)
		botCfg.RLock()
		restart := botCfg.restart
		botCfg.RUnlock()
		if restart {
			Log(robot.Info, "Restarting...")
		}
		done <- restart
		// NOTE!! Black Magic Ahead - for some reason, the read on the done channel
		// keeps blocking without this close.
		close(done)
	}(botCfg.Connector, botCfg.stop, botCfg.done)
	botCfg.RUnlock()
	return botCfg.done
}

// stop is called whenever the robot needs to shut down gracefully. All callers
// should lock the bot and check the value of botCfg.shuttingDown; see
// builtins.go.
func stop() {
	botCfg.RLock()
	pr := botCfg.pluginsRunning
	stop := botCfg.stop
	botCfg.RUnlock()
	Log(robot.Debug, "stop called with %d plugins running", pr)
	botCfg.Wait()
	brainQuit()
	close(stop)
}
