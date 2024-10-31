package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/lnxjedi/gopherbot/robot"
)

/* conf.go - methods and types for reading and storing json configuration */

var protocolConfig, brainConfig, historyConfig json.RawMessage

var robotConfigFileName = "robot.yaml"

// ConfigLoader defines 'bot configuration, and is read from conf/robot.yaml
// Digested content ends up in currentCfg, see bot_process.go.
// ConfigLoader represents the structure of robot.yaml for validation.
type ConfigLoader struct {
	AdminContact         string                  `yaml:"AdminContact"`         // Contact info for whomever administers the robot
	MailConfig           botMailer               `yaml:"MailConfig"`           // Configuration for sending email
	Protocol             string                  `yaml:"Protocol"`             // Name of the connector protocol to use, e.g., "slack"
	ProtocolConfig       json.RawMessage         `yaml:"ProtocolConfig"`       // Protocol-specific configuration, for unmarshalling arbitrary config
	BotInfo              *UserInfo               `yaml:"BotInfo"`              // Information about the robot
	UserRoster           []UserInfo              `yaml:"UserRoster"`           // List of users and related attributes
	ChannelRoster        []ChannelInfo           `yaml:"ChannelRoster"`        // List of channels mapping names to IDs
	Brain                string                  `yaml:"Brain"`                // Type of Brain to use
	BrainConfig          json.RawMessage         `yaml:"BrainConfig"`          // Brain-specific configuration, for unmarshalling arbitrary config
	EncryptBrain         bool                    `yaml:"EncryptBrain"`         // Whether the brain should be encrypted
	EncryptionKey        string                  `yaml:"EncryptionKey"`        // Used to decrypt the "real" encryption key
	HistoryProvider      string                  `yaml:"HistoryProvider"`      // Name of provider to use for storing and retrieving job/plugin histories
	HistoryConfig        json.RawMessage         `yaml:"HistoryConfig"`        // History provider-specific configuration
	WorkSpace            string                  `yaml:"WorkSpace"`            // Read/Write area the robot uses to do work
	DefaultElevator      string                  `yaml:"DefaultElevator"`      // Elevator plugin for ElevatedCommands and ElevateImmediateCommands
	DefaultAuthorizer    string                  `yaml:"DefaultAuthorizer"`    // Authorizer plugin for AuthorizedCommands, or when AuthorizeAllCommands = true
	DefaultMessageFormat string                  `yaml:"DefaultMessageFormat"` // How the robot formats outgoing messages; default: Raw
	DefaultAllowDirect   bool                    `yaml:"DefaultAllowDirect"`   // Whether plugins are available in a DM by default
	IgnoreUnlistedUsers  bool                    `yaml:"IgnoreUnlistedUsers"`  // Drop all messages from ID not in the UserRoster
	DefaultChannels      []string                `yaml:"DefaultChannels"`      // Channels where plugins are active by default, e.g., ["general", "random"]
	IgnoreUsers          []string                `yaml:"IgnoreUsers"`          // Users the bot never talks to - like other bots
	JoinChannels         []string                `yaml:"JoinChannels"`         // Channels the bot should join on login (not supported by all protocols)
	DefaultJobChannel    string                  `yaml:"DefaultJobChannel"`    // Where job status is posted by default
	TimeZone             string                  `yaml:"TimeZone"`             // For evaluating the hour in a job schedule
	ExternalJobs         map[string]TaskSettings `yaml:"ExternalJobs"`         // List of available jobs; config in conf/jobs/<jobname>.yaml
	ExternalPlugins      map[string]TaskSettings `yaml:"ExternalPlugins"`      // List of non-Go plugins to load; config in conf/plugins/<plugname>.yaml
	ExternalTasks        map[string]TaskSettings `yaml:"ExternalTasks"`        // List executables for pipeline addition (not as starters)
	GoJobs               map[string]TaskSettings `yaml:"GoJobs"`               // Settings for Go jobs; config in conf/jobs/<jobname>.yaml
	GoPlugins            map[string]TaskSettings `yaml:"GoPlugins"`            // Settings for Go plugins; config in conf/plugins/<plugname>.yaml
	GoTasks              map[string]TaskSettings `yaml:"GoTasks"`              // Settings for Go tasks
	NameSpaces           map[string]TaskSettings `yaml:"NameSpaces"`           // Namespaces for shared parameters & memory sharing
	ParameterSets        map[string]TaskSettings `yaml:"ParameterSets"`        // Named sets of parameters, e.g., GITHUB_TOKEN used multiple places
	ScheduledJobs        []ScheduledTask         `yaml:"ScheduledJobs"`        // See tasks.go
	AdminUsers           []string                `yaml:"AdminUsers"`           // List of users with access to administrative commands
	Alias                string                  `yaml:"Alias"`                // One-character alias for commands directed at the bot, e.g., ';open the pod bay doors'
	LocalPort            int                     `yaml:"LocalPort"`            // Port number for localhost listening for CLI plugins
	LogLevel             string                  `yaml:"LogLevel"`             // Initial log level, modifiable by plugins. Options: "trace," "debug," "info," "warn," "error"
}

// UserInfo is listed in the UserRoster of robot.yaml to provide:
// - Attributes and info that might not be provided by the connector:
//   - Mapping of protocol internal ID to username
//   - Additional user attributes such as first / last name, email, etc.
//
// - Additional information needed by bot internals
//   - BotUser flag
type UserInfo struct {
	UserName     string `yaml:"UserName"`  // Name that refers to the user in bot config files
	UserID       string `yaml:"UserID"`    // Unique/persistent ID given to the user by the connector
	Email        string `yaml:"Email"`     // For Get*Attribute()
	Phone        string `yaml:"Phone"`     // For Get*Attribute()
	FullName     string `yaml:"FullName"`  // For Get*Attribute()
	FirstName    string `yaml:"FirstName"` // For Get*Attribute()
	LastName     string `yaml:"LastName"`  // For Get*Attribute()
	protoMention string `yaml:"-"`         // Robot only, @(mention) string
	BotUser      bool   `yaml:"BotUser"`   // These users aren't checked against MessageMatchers/ambient messages and never fall-through to "catchalls"
}

// ChannelInfo maps channel IDs to channel names when the connector doesn't
// provide a sensible name for use in configuration files.
type ChannelInfo struct {
	ChannelName, ChannelID string // human-readable and protocol-internal channel representations
}

type userChanMaps struct {
	userID    map[string]*UserInfo    // Current map of userID to UserInfo struct
	user      map[string]*UserInfo    // Current map of username to UserInfo struct
	channelID map[string]*ChannelInfo // Current map of channel ID to ChannelInfo struct
	channel   map[string]*ChannelInfo // Current map of channel name to ChannelInfo struct
}

var currentUCMaps = struct {
	ucmap *userChanMaps // pointer to current struct
	sync.Mutex
}{
	nil,
	sync.Mutex{},
}

// Protects the bot config
var confLock sync.RWMutex
var config *ConfigLoader

// loadConfig loads the 'bot's yaml configuration files.
func loadConfig(preConnect bool) error {
	raiseThreadPriv("loading configuration")
	var loglevel robot.LogLevel
	if preConnect {
		Log(robot.Info, "Loading initial pre-connection configuration")
	} else {
		Log(robot.Info, "Loading full post-connect configuration")
	}
	newconfig := &ConfigLoader{}
	newconfig.ExternalJobs = make(map[string]TaskSettings)
	newconfig.ExternalPlugins = make(map[string]TaskSettings)
	newconfig.ExternalTasks = make(map[string]TaskSettings)
	configload := make(map[string]json.RawMessage)
	processed := &configuration{}

	if err := getConfigFile(robotConfigFileName, true, configload); err != nil {
		return fmt.Errorf("Loading configuration file: %v", err)
	}

	explicitDefaultAllowDirect := false

	for key, value := range configload {
		var strval string
		var sarrval []string
		var urval []UserInfo
		var bival *UserInfo
		var crval []ChannelInfo
		var tval map[string]TaskSettings
		var stval []ScheduledTask
		var mailval botMailer
		var boolval bool
		var intval int
		var val interface{}
		skip := false
		switch key {
		case "AdminContact", "Email", "Protocol", "Brain", "EncryptionKey", "HistoryProvider", "WorkSpace", "DefaultJobChannel", "DefaultElevator", "DefaultAuthorizer", "DefaultMessageFormat", "Name", "Alias", "LogLevel", "TimeZone":
			val = &strval
		case "DefaultAllowDirect", "EncryptBrain", "IgnoreUnlistedUsers":
			val = &boolval
		case "BotInfo":
			val = &bival
		case "UserRoster":
			val = &urval
		case "ChannelRoster":
			val = &crval
		case "LocalPort":
			val = &intval
		case "ExternalJobs", "ExternalPlugins", "ExternalTasks", "GoJobs", "GoPlugins", "GoTasks", "NameSpaces", "ParameterSets":
			val = &tval
		case "ScheduledJobs":
			val = &stval
		case "DefaultChannels", "IgnoreUsers", "JoinChannels", "AdminUsers":
			val = &sarrval
		case "MailConfig":
			val = &mailval
		case "ProtocolConfig", "BrainConfig", "HistoryConfig":
			skip = true
		default:
			err := fmt.Errorf("Invalid configuration key in %s: %s", robotConfigFileName, key)
			Log(robot.Error, err.Error())
			return err
		}
		if !skip {
			if err := json.Unmarshal(value, val); err != nil {
				err = fmt.Errorf("Unmarshalling bot config value \"%s\": %v", key, err)
				Log(robot.Error, err.Error())
				return err
			}
		}
		switch key {
		case "AdminContact":
			newconfig.AdminContact = *(val.(*string))
		case "BotInfo":
			newconfig.BotInfo = *(val.(**UserInfo))
		case "MailConfig":
			newconfig.MailConfig = *(val.(*botMailer))
		case "Protocol":
			newconfig.Protocol = *(val.(*string))
		case "ProtocolConfig":
			newconfig.ProtocolConfig = value
		case "Brain":
			newconfig.Brain = *(val.(*string))
		case "EncryptionKey":
			newconfig.EncryptionKey = *(val.(*string))
		case "BrainConfig":
			newconfig.BrainConfig = value
		case "HistoryProvider":
			newconfig.HistoryProvider = *(val.(*string))
		case "HistoryConfig":
			newconfig.HistoryConfig = value
		case "WorkSpace":
			newconfig.WorkSpace = *(val.(*string))
		case "DefaultJobChannel":
			newconfig.DefaultJobChannel = *(val.(*string))
		case "DefaultElevator":
			newconfig.DefaultElevator = *(val.(*string))
		case "DefaultAuthorizer":
			newconfig.DefaultAuthorizer = *(val.(*string))
		case "DefaultMessageFormat":
			newconfig.DefaultMessageFormat = *(val.(*string))
		case "UserRoster":
			newconfig.UserRoster = *(val.(*[]UserInfo))
		case "ChannelRoster":
			newconfig.ChannelRoster = *(val.(*[]ChannelInfo))
		case "DefaultAllowDirect":
			newconfig.DefaultAllowDirect = *(val.(*bool))
			explicitDefaultAllowDirect = true
		case "DefaultChannels":
			newconfig.DefaultChannels = *(val.(*[]string))
		case "IgnoreUsers":
			newconfig.IgnoreUsers = *(val.(*[]string))
		case "JoinChannels":
			newconfig.JoinChannels = *(val.(*[]string))
		case "EncryptBrain":
			newconfig.EncryptBrain = *(val.(*bool))
		case "IgnoreUnlistedUsers":
			newconfig.IgnoreUnlistedUsers = *(val.(*bool))
		case "ExternalPlugins":
			newconfig.ExternalPlugins = *(val.(*map[string]TaskSettings))
		case "ExternalJobs":
			newconfig.ExternalJobs = *(val.(*map[string]TaskSettings))
		case "ExternalTasks":
			newconfig.ExternalTasks = *(val.(*map[string]TaskSettings))
		case "GoPlugins":
			newconfig.GoPlugins = *(val.(*map[string]TaskSettings))
		case "GoJobs":
			newconfig.GoJobs = *(val.(*map[string]TaskSettings))
		case "GoTasks":
			newconfig.GoTasks = *(val.(*map[string]TaskSettings))
		case "NameSpaces":
			newconfig.NameSpaces = *(val.(*map[string]TaskSettings))
		case "ParameterSets":
			newconfig.ParameterSets = *(val.(*map[string]TaskSettings))
		case "ScheduledJobs":
			newconfig.ScheduledJobs = *(val.(*[]ScheduledTask))
		case "AdminUsers":
			newconfig.AdminUsers = *(val.(*[]string))
		case "Alias":
			newconfig.Alias = *(val.(*string))
		case "LocalPort":
			newconfig.LocalPort = *(val.(*int))
		case "LogLevel":
			newconfig.LogLevel = *(val.(*string))
		case "TimeZone":
			newconfig.TimeZone = *(val.(*string))
		}
	}

	// Leave loglevel at Warn for CLI operations
	if !cliOp {
		loglevel = logStrToLevel(newconfig.LogLevel)
		setLogLevel(loglevel)
	}

	processed.ignoreUnlistedUsers = newconfig.IgnoreUnlistedUsers
	if newconfig.Protocol != "" {
		processed.protocol = newconfig.Protocol
	} else {
		return fmt.Errorf("Protocol not specified in %s", robotConfigFileName)
	}
	if newconfig.Brain != "" {
		processed.brainProvider = newconfig.Brain
	}
	if newconfig.BrainConfig != nil {
		brainConfig = newconfig.BrainConfig
	}
	if newconfig.HistoryProvider != "" {
		processed.historyProvider = newconfig.HistoryProvider
	}
	if newconfig.HistoryConfig != nil {
		historyConfig = newconfig.HistoryConfig
	}

	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		if !strings.ContainsRune(string(aliases+escapeAliases), alias) {
			return fmt.Errorf("Invalid alias specified, ignoring. Must be one of: %s%s", escapeAliases, aliases)
		}
		processed.alias = alias
	}

	if len(newconfig.DefaultMessageFormat) == 0 {
		processed.defaultMessageFormat = robot.Raw
	} else {
		processed.defaultMessageFormat = setFormat(newconfig.DefaultMessageFormat)
	}

	if explicitDefaultAllowDirect {
		processed.defaultAllowDirect = newconfig.DefaultAllowDirect
	} else {
		processed.defaultAllowDirect = true // rare case of defaulting to true
	}

	if newconfig.AdminContact != "" {
		processed.adminContact = newconfig.AdminContact
	}

	if newconfig.TimeZone != "" {
		tz, err := time.LoadLocation(newconfig.TimeZone)
		if err == nil {
			Log(robot.Info, "Set timezone: %s", tz)
			processed.timeZone = tz
		} else {
			Log(robot.Error, "Parsing time zone '%s', using local time; error: %v", newconfig.TimeZone, err)
			processed.timeZone = nil
		}
	}

	if newconfig.BotInfo != nil {
		// Note that connector-supplied values are copied
		// when processed becomes current.
		processed.botinfo = *newconfig.BotInfo
	}
	processed.mailConf = newconfig.MailConfig

	if newconfig.DefaultJobChannel != "" {
		processed.defaultJobChannel = newconfig.DefaultJobChannel
	}

	if newconfig.DefaultElevator != "" {
		processed.defaultElevator = newconfig.DefaultElevator
	}

	if newconfig.DefaultAuthorizer != "" {
		processed.defaultAuthorizer = newconfig.DefaultAuthorizer
	}

	if newconfig.AdminUsers != nil {
		processed.adminUsers = newconfig.AdminUsers
	} else {
		processed.adminUsers = []string{}
	}
	if newconfig.DefaultChannels != nil {
		processed.plugChannels = newconfig.DefaultChannels
	}
	if newconfig.ExternalPlugins != nil {
		et := make([]TaskSettings, 0)
		for name, task := range newconfig.ExternalPlugins {
			if task.Disabled {
				continue
			}
			task.Name = name
			if task.Privileged == nil {
				p := false
				task.Privileged = &p
			}
			et = append(et, task)
		}
		processed.externalPlugins = et
	}
	if newconfig.ExternalJobs != nil {
		et := make([]TaskSettings, 0)
		for name, task := range newconfig.ExternalJobs {
			if task.Disabled {
				continue
			}
			task.Name = name
			if task.Privileged == nil {
				p := true
				task.Privileged = &p
			}
			et = append(et, task)
		}
		processed.externalJobs = et
	}
	if newconfig.ExternalTasks != nil {
		et := make([]TaskSettings, 0)
		for name, task := range newconfig.ExternalTasks {
			if task.Disabled {
				continue
			}
			task.Name = name
			if task.Privileged == nil {
				p := false
				task.Privileged = &p
			}
			et = append(et, task)
		}
		processed.externalTasks = et
	}
	// NOTE on Go tasks - we can't just skip a disabled task, since they're
	// enabled by default. Disabled: true needs to pass through so it's disabled
	// in taskconf.go
	if newconfig.GoTasks != nil {
		gt := make([]TaskSettings, 0, len(newconfig.GoTasks))
		for name, task := range newconfig.GoTasks {
			task.Name = name
			gt = append(gt, task)
		}
		processed.goTasks = gt
	}
	if newconfig.GoPlugins != nil {
		gt := make([]TaskSettings, 0, len(newconfig.GoPlugins))
		for name, task := range newconfig.GoPlugins {
			task.Name = name
			gt = append(gt, task)
		}
		processed.goPlugins = gt
	}
	if newconfig.GoJobs != nil {
		gt := make([]TaskSettings, 0, len(newconfig.GoJobs))
		for name, task := range newconfig.GoJobs {
			task.Name = name
			gt = append(gt, task)
		}
		processed.goJobs = gt
	}
	if newconfig.NameSpaces != nil {
		ns := make([]TaskSettings, 0, len(newconfig.NameSpaces))
		for name, nameSpace := range newconfig.NameSpaces {
			nameSpace.Name = name
			ns = append(ns, nameSpace)
		}
		processed.nsList = ns
	}
	if newconfig.ParameterSets != nil {
		ps := make([]TaskSettings, 0, len(newconfig.ParameterSets))
		for name, parameterSet := range newconfig.ParameterSets {
			parameterSet.Name = name
			ps = append(ps, parameterSet)
		}
		processed.psList = ps
	}
	st := make([]ScheduledTask, 0, len(newconfig.ScheduledJobs))
	for _, s := range newconfig.ScheduledJobs {
		if len(s.Name) == 0 || len(s.Schedule) == 0 {
			Log(robot.Error, "Zero-length Name (%s) or Schedule (%s) in ScheduledTask, skipping", s.Name, s.Schedule)
		} else {
			st = append(st, s)
		}
	}
	processed.ScheduledJobs = st
	if newconfig.IgnoreUsers != nil {
		processed.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		processed.joinChannels = newconfig.JoinChannels
	}

	ucmaps := userChanMaps{
		make(map[string]*UserInfo),
		make(map[string]*UserInfo),
		make(map[string]*ChannelInfo),
		make(map[string]*ChannelInfo),
	}
	usermap := make(map[string]string)
	if len(newconfig.UserRoster) > 0 {
		for i, user := range newconfig.UserRoster {
			if len(user.UserName) == 0 || len(user.UserID) == 0 {
				Log(robot.Error, "One of Username/UserID empty (%s/%s), ignoring", user.UserName, user.UserID)
			} else {
				u := &newconfig.UserRoster[i]
				ucmaps.user[u.UserName] = u
				ucmaps.userID[u.UserID] = u
				usermap[u.UserName] = u.UserID
			}
		}
	}
	if len(newconfig.ChannelRoster) > 0 {
		for i, ch := range newconfig.ChannelRoster {
			if len(ch.ChannelName) == 0 || len(ch.ChannelID) == 0 {
				Log(robot.Error, "One of ChannelName/ChannelID empty (%s/%s), ignoring", ch.ChannelName, ch.ChannelID)
			} else {
				c := &newconfig.ChannelRoster[i]
				ucmaps.channel[c.ChannelName] = c
				ucmaps.channelID[c.ChannelID] = c
			}
		}
	}
	currentUCMaps.Lock()
	currentUCMaps.ucmap = &ucmaps
	currentUCMaps.Unlock()

	h := handler{}
	if len(newconfig.WorkSpace) > 0 {
		if err := h.GetDirectory(newconfig.WorkSpace); err == nil {
			processed.workSpace = newconfig.WorkSpace
			Log(robot.Debug, "Setting workspace directory to '%s'", processed.workSpace)
		} else {
			Log(robot.Error, "Getting WorkSpace directory '%s', using '%s': %v", newconfig.WorkSpace, configPath, err)
		}
	}

	// Items only read at start-up, before multi-threaded
	if preConnect {
		if newconfig.ProtocolConfig != nil {
			protocolConfig = newconfig.ProtocolConfig
		}

		if newconfig.EncryptBrain {
			encryptBrain = true
		}
		if newconfig.EncryptionKey != "" {
			processed.encryptionKey = newconfig.EncryptionKey
			newconfig.EncryptionKey = "XXXXXX" // too short to be valid anyway
		}
		if newconfig.LocalPort != 0 {
			processed.port = fmt.Sprintf("%d", newconfig.LocalPort)
		} else {
			processed.port = "0"
		}
		if len(newconfig.HistoryProvider) == 0 {
			newconfig.HistoryProvider = "mem"
		}
		var hprovider func(robot.Handler) robot.HistoryProvider
		var ok bool
		if !cliOp { // CLI operations don't need a real history
			if hprovider, ok = historyProviders[newconfig.HistoryProvider]; !ok {
				Log(robot.Error, "No provider registered for history type: \"%s\", falling back to 'mem'", processed.historyProvider)
				newconfig.HistoryProvider = "mem"
				hprovider = historyProviders["mem"]
			}
			hp := hprovider(handler{})
			interfaces.history = hp
			if newconfig.HistoryProvider != "mem" {
				// Initialize the memory provider as a last-ditch fallback
				mhprovider(handler{})
			}
		}
	} else {
		if len(usermap) > 0 {
			interfaces.SetUserMap(usermap)
		}
		// We should never dump the brain key
		newconfig.EncryptionKey = "XXXXXX"
		// initJobs need to run before post-connect loadTaskConfig
		initJobs()
	}

	newList, err := loadTaskConfig(processed, preConnect)
	if err != nil {
		return err
	}

	// Configuration successfully loaded, apply changes

	// Note we always take the locks on global values regardless
	// of preConnect.

	// Update structs supplied to "dump robot"
	// Note that dump commands are only allowed for the
	// terminal connector.
	confLock.Lock()
	config = newconfig
	confLock.Unlock()

	currentCfg.Lock()
	processed.botinfo.UserID = currentCfg.botinfo.UserID
	processed.botinfo.protoMention = currentCfg.botinfo.protoMention
	currentCfg.configuration = processed
	currentCfg.taskList = newList
	currentCfg.Unlock()

	if !preConnect {
		updateRegexes()
		scheduleTasks()
		initializePlugins()
	}

	return nil
}
