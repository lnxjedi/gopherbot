// Package bot provides the internal machinery for most of Gopherbot.
package bot

/* bot.go defines core data structures and public methods for startup.
   handler.go has the methods for callbacks from the connector, */

import (
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
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

// Interfaces to external stuff, items should be set while single-threaded and never change
var interfaces struct {
	robot.Connector                       // Connector interface, implemented by each specific protocol
	brain           robot.SimpleBrain     // Interface for robot to Store and Retrieve data
	history         robot.HistoryProvider // Provider for storing and retrieving job / plugin histories
}

var done = make(chan bool)              // shutdown channel, true to restart
var stopConnector = make(chan struct{}) // stop channel for stopping the connector

// internal state tracking
var state struct {
	shuttingDown     bool // to prevent new plugins from starting
	restart          bool // indicate stop and restart vs. stop only, for bootstrapping
	pipelinesRunning int  // a count of how many plugins are currently running
	sync.WaitGroup        // for keeping track of running plugins
	sync.RWMutex          // for safe updating of bot data structures
}

// regexes the bot uses to determine if it's being spoken to
var regexes struct {
	preRegex  *regexp.Regexp // regex for matching prefixed commands, e.g. "Gort, drop your weapon"
	postRegex *regexp.Regexp // regex for matching, e.g. "open the pod bay doors, hal"
	bareRegex *regexp.Regexp // regex for matching the robot's bare name, if you forgot it in the previous command
	sync.RWMutex
}

// configuration struct holds all the interal data relevant to the Bot. Most of it is digested
// and populated by loadConfig.
type configuration struct {
	adminUsers           []string            // List of users with access to administrative commands
	alias                rune                // single-char alias for addressing the bot
	botinfo              UserInfo            // robot's name, ID, email, etc.
	adminContact         string              // who to contact for problems with the bot
	mailConf             botMailer           // configuration to use when sending email
	ignoreUsers          []string            // list of users to never listen to, like other bots
	joinChannels         []string            // list of channels to join
	defaultAllowDirect   bool                // whether plugins are available in DM by default
	defaultMessageFormat robot.MessageFormat // Raw unless set to Variable or Fixed
	plugChannels         []string            // list of channels where plugins are available by default
	protocol             string              // Name of the protocol, e.g. "slack"
	brainProvider        string              // Type of Brain provider to use
	encryptionKey        string              // Key for encrypting data (unlocks "real" key in brain)
	historyProvider      string              // Name of the history provider to use
	workSpace            string              // Read/Write directory where the robot does work
	defaultElevator      string              // Plugin name for performing elevation
	defaultAuthorizer    string              // Plugin name for performing authorization
	externalPlugins      []TaskSettings      // List of external plugins to load
	externalJobs         []TaskSettings      // List of external jobs to load
	externalTasks        []TaskSettings      // List of external tasks to load
	goPlugins            []TaskSettings      // Settings for goPlugins: Name(match), Description, NameSpace, Parameters, Disabled
	goJobs               []TaskSettings      // Settings for goJobs: Name(match), Description, NameSpace, Parameters, Disabled
	goTasks              []TaskSettings      // Settings for goTasks: Name(match), Description, NameSpace, Parameters, Disabled
	nsList               []TaskSettings      // loaded NameSpaces for shared parameters
	loadableModules      []LoadableModule    // List of loadable modules to load
	ScheduledJobs        []ScheduledTask     // List of scheduled tasks
	port                 string              // Configured localhost port to listen on, or 0 for first open
	timeZone             *time.Location      // for forcing the TimeZone, Unix only
	defaultJobChannel    string              // where job statuses will post if not otherwise specified
}

// The current configuration and task list
var currentCfg = struct {
	*configuration
	*taskList
	sync.RWMutex
}{
	configuration: &configuration{},
	taskList: &taskList{
		t:          []interface{}{struct{}{}}, // initialize 0 to "nothing", for namespaces only
		nameMap:    make(map[string]int),
		idMap:      make(map[string]int),
		nameSpaces: make(map[string]NameSpace),
	},
	RWMutex: sync.RWMutex{},
}

var listening bool    // for tests where initBot runs multiple times
var listenPort string // actual listening port

// initBot sets up the global robot; when cli is false it also loads configuration.
// cli indicates that a CLI command is being processed, as opposed to actually running
// a robot.
func initBot(cpath, epath string, logger *log.Logger) {
	// Seed the pseudo-random number generator, for plugin IDs, RandomString, etc.
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Initialize current config with an empty struct (to be loaded)
	currentCfg.configuration = &configuration{}

	// Only true with test suite
	if logger != nil {
		botLogger.l = logger
	}

	var err error
	homePath, err = os.Getwd()
	if err != nil {
		Log(robot.Warn, "Unable to get cwd")
	}
	h := handler{}
	if err := h.GetDirectory(cpath); err != nil {
		Log(robot.Fatal, "Unable to get/create config path: %s", cpath)
	}
	configPath = cpath
	installPath = epath

	state.shuttingDown = false

	if cliOp {
		setLogLevel(robot.Warn)
	}

	encryptionInitialized := initCrypt()

	if err := loadConfig(true); err != nil {
		Log(robot.Fatal, "Loading initial configuration: %v", err)
	}
	os.Unsetenv(keyEnv)

	if cliOp {
		if fileLog {
			setLogLevel(robot.Debug)
		} else {
			setLogLevel(robot.Warn)
		}
	}

	// loadModules for go loadable modules; a no-op for static builds
	loadModules()

	// All pluggables registered, ok to stop registrations
	stopRegistrations = true

	if len(currentCfg.brainProvider) > 0 {
		if bprovider, ok := brains[currentCfg.brainProvider]; !ok {
			Log(robot.Fatal, "No provider registered for brain: \"%s\"", currentCfg.brainProvider)
		} else {
			brain := bprovider(handle)
			interfaces.brain = brain
			Log(robot.Info, "Initialized brain provider '%s'", currentCfg.brainProvider)
		}
	} else {
		bprovider, _ := brains["mem"]
		interfaces.brain = bprovider(handle)
		Log(robot.Error, "No brain configured, falling back to default 'mem' brain - no memories will persist")
	}
	if !encryptionInitialized && len(currentCfg.encryptionKey) > 0 {
		if initializeEncryptionFromBrain(currentCfg.encryptionKey) {
			Log(robot.Info, "Successfully initialized encryption from configured key")
			encryptionInitialized = true
		} else {
			Log(robot.Error, "Failed to initialize brain encryption with configured EncryptionKey")
		}
	}
	if encryptBrain && !encryptionInitialized {
		Log(robot.Warn, "Brain encryption specified but not initialized; use 'initialize brain <key>' to initialize the encrypted brain interactively")
	}

	// cli commands don't need an http listener
	if cliOp {
		return
	}

	if !listening {
		listening = true
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%s", currentCfg.port))
		if err != nil {
			Log(robot.Fatal, "Listening on tcp4 port 127.0.0.1:%s: %v", currentCfg.port, err)
		}
		listenPort = listener.Addr().String()
		go func() {
			raiseThreadPriv("http handler")
			http.Handle("/json", handle)
			Log(robot.Info, "Listening for external plugin connections on http://%s", listenPort)
			Log(robot.Fatal, "Error serving '/json': %s", http.Serve(listener, nil))
		}()
	}
}

// set connector sets the connector, which should already be initialized
func setConnector(c robot.Connector) {
	interfaces.Connector = c
}

var keyEnv = "GOPHER_ENCRYPTION_KEY"

func initCrypt() bool {
	// Initialize encryption (new style for v2)
	keyFile := filepath.Join(configPath, encryptedKeyFile)
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
			if len(currentCfg.encryptionKey) == 0 {
				// No encryptionKey in config, create new-style key
				bk := make([]byte, 32)
				_, err := crand.Read(bk)
				if err != nil {
					Log(robot.Error, "Generating new random encryption key: %v", err)
					return false
				}
				bek, err := encrypt(bk, ik)
				if err != nil {
					Log(robot.Error, "Encrypting new random key: %v", err)
					return false
				}
				beks := base64.StdEncoding.EncodeToString(bek)
				err = ioutil.WriteFile(keyFile, []byte(beks), 0444)
				if err != nil {
					Log(robot.Error, "Writing out generated key: %v", err)
					return false
				}
				Log(robot.Info, "Successfully wrote new binary encryption key to '%s'", keyFile)
				cryptKey.key = bk
				cryptKey.initialized = true
				encryptionInitialized = true
				return true
			}
		}
		os.Unsetenv(keyEnv)
	} else {
		Log(robot.Warn, "GOPHER_ENCRYPTION_KEY not set in environment")
	}
	return encryptionInitialized
}

// run starts all the loops and returns a channel that closes when the robot
// shuts down. It should return after the connector loop has started and
// plugins are initialized.
func run() {
	// Start the brain loop
	go runBrain()

	var cl []string
	cl = append(cl, currentCfg.joinChannels...)
	cl = append(cl, currentCfg.plugChannels...)
	cl = append(cl, currentCfg.defaultJobChannel)
	jc := make(map[string]bool)
	for _, channel := range cl {
		if _, ok := jc[channel]; !ok {
			jc[channel] = true
			interfaces.JoinChannel(channel)
		}
	}

	// signal handler
	sigBreak := make(chan struct{})
	go sigHandle(sigBreak)

	// connector loop
	go func(conn robot.Connector, sigBreak chan<- struct{}) {
		raiseThreadPriv("connector loop")
		conn.Run(stopConnector)
		sigBreak <- struct{}{}
		state.RLock()
		restart := state.restart
		state.RUnlock()
		if restart {
			Log(robot.Info, "Restarting...")
		}
		done <- restart
	}(interfaces.Connector, sigBreak)

	loadConfig(false)
}

// stop is called whenever the robot needs to shut down gracefully. All callers
// should lock the bot and check the value of botCfg.shuttingDown; see
// builtins.go.
func stop() {
	state.RLock()
	pr := state.pipelinesRunning
	state.RUnlock()
	Log(robot.Debug, "stop called with %d plugins running", pr)
	state.Wait()
	brainQuit()
	stopConnector <- struct{}{}
}
