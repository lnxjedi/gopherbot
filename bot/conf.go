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

// ConfigLoader defines 'bot configuration, and is read from conf/gopherbot.yaml
type ConfigLoader struct {
	AdminContact         string                    // Contact info for whomever administers the robot
	MailConfig           botMailer                 // configuration for sending email
	Protocol             string                    // Name of the connector protocol to use, e.g. "slack"
	ProtocolConfig       json.RawMessage           // Protocol-specific configuration, type for unmarshalling arbitrary config
	BotInfo              *UserInfo                 // Information about the robot
	UserRoster           []UserInfo                // List of users and related attributes
	ChannelRoster        []ChannelInfo             // List of channels mapping names to IDs
	Brain                string                    // Type of Brain to use
	BrainConfig          json.RawMessage           // Brain-specific configuration, type for unmarshalling arbitrary config
	EncryptBrain         bool                      // Whether the brain should be encrypted
	EncryptionKey        string                    // used to decrypt the "real" encryption key
	HistoryProvider      string                    // Name of provider to use for storing and retrieving job/plugin histories
	HistoryConfig        json.RawMessage           // History provider specific configuration
	WorkSpace            string                    // Read/Write area the robot uses to do work
	DefaultElevator      string                    // Elevator plugin to use by default for ElevatedCommands and ElevateImmediateCommands
	DefaultAuthorizer    string                    // Authorizer plugin to use by default for AuthorizedCommands, or when AuthorizeAllCommands = true
	DefaultMessageFormat string                    // How the robot should format outgoing messages unless told otherwise; default: Raw
	DefaultAllowDirect   bool                      // Whether plugins are available in a DM by default
	DefaultChannels      []string                  // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers          []string                  // Users the 'bot never talks to - like other bots
	JoinChannels         []string                  // Channels the 'bot should join when it logs in (not supported by all protocols)
	DefaultJobChannel    string                    // Where job status is posted by default
	TimeZone             string                    // For evaluating the hour in a job schedule
	ExternalJobs         map[string]TaskSettings   // list of available jobs; config in conf/jobs/<jobname>.yaml
	ExternalPlugins      map[string]TaskSettings   // List of non-Go plugins to load; config in conf/plugins/<plugname>.yaml
	ExternalTasks        map[string]TaskSettings   // List executables that can be added to a pipeline (but can't start one)
	GoJobs               map[string]TaskSettings   // settings for go jobs; config in conf/jobs/<jobname>.yaml
	GoPlugins            map[string]TaskSettings   // settings for go plugins; config in conf/plugins/<plugname>.yaml
	GoTasks              map[string]TaskSettings   // settings for go tasks
	NameSpaces           map[string]TaskSettings   // namespaces for shared parameters & memory sharing
	LoadableModules      map[string]LoadableModule // List of loadable modules to load
	ScheduledJobs        []ScheduledTask           // see tasks.go
	AdminUsers           []string                  // List of users who can access administrative commands
	Alias                string                    // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort            int                       // Port number for listening on localhost, for CLI plugins
	LogLevel             string                    // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
}

// Repository represents a buildable git repository, for CI/CD
type Repository struct {
	Type         string // task extending the namespace needs to match for parameters
	CloneURL     string
	Dependencies []string    // List of repositories this one depends on; changes to a dependency trigger a build
	KeepHistory  int         // How many job logs to keep for this repo
	Parameters   []Parameter // per-repository parameters
}

// UserInfo is listed in the UserRoster of gopherbot.yaml to provide:
// - Attributes and info that might not be provided by the connector:
//   - Mapping of protocol internal ID to username
//   - Additional user attributes such as first / last name, email, etc.
// - Additional information needed by bot internals
//   - BotUser flag
type UserInfo struct {
	UserName            string // name that refers to the user in bot config files
	UserID              string // unique/persistent ID given to the user by the connector
	Email, Phone        string // for Get*Attribute()
	FullName            string // for Get*Attribute()
	FirstName, LastName string // for Get*Attribute()
	protoMention        string // robot only, @(mention) string
	BotUser             bool   // these users aren't checked against MessageMatchers / ambient messages, and never fall-through to "catchalls"
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

// Protects the bot config and list of repositories
var confLock sync.RWMutex
var config *ConfigLoader
var repositories map[string]Repository

// loadConfig loads the 'bot's yaml configuration files.
func (c *botContext) loadConfig(preConnect bool) error {
	var loglevel robot.LogLevel
	newconfig := &ConfigLoader{}
	newconfig.ExternalJobs = make(map[string]TaskSettings)
	newconfig.ExternalPlugins = make(map[string]TaskSettings)
	newconfig.ExternalTasks = make(map[string]TaskSettings)
	configload := make(map[string]json.RawMessage)

	if err := c.getConfigFile("gopherbot.yaml", "", true, configload); err != nil {
		return fmt.Errorf("Loading configuration file: %v", err)
	}

	reporaw := make(map[string]json.RawMessage)
	c.getConfigFile("repositories.yaml", "", false, reporaw)
	repolist := make(map[string]Repository)
	for k, repojson := range reporaw {
		if strings.ContainsRune(k, ':') {
			Log(robot.Error, "Invalid repository '%s' contains ':', ignoring", k)
		} else {
			var repository Repository
			json.Unmarshal(repojson, &repository)
			repolist[k] = repository
		}
	}

	explicitDefaultAllowDirect := false

	for key, value := range configload {
		var strval string
		var sarrval []string
		var urval []UserInfo
		var bival *UserInfo
		var crval []ChannelInfo
		var tval map[string]TaskSettings
		var mval map[string]LoadableModule
		var stval []ScheduledTask
		var mailval botMailer
		var boolval bool
		var intval int
		var val interface{}
		skip := false
		switch key {
		case "AdminContact", "Email", "Protocol", "Brain", "EncryptionKey", "HistoryProvider", "WorkSpace", "DefaultJobChannel", "DefaultElevator", "DefaultAuthorizer", "DefaultMessageFormat", "Name", "Alias", "LogLevel", "TimeZone":
			val = &strval
		case "DefaultAllowDirect", "EncryptBrain":
			val = &boolval
		case "BotInfo":
			val = &bival
		case "UserRoster":
			val = &urval
		case "ChannelRoster":
			val = &crval
		case "LocalPort":
			val = &intval
		case "ExternalJobs", "ExternalPlugins", "ExternalTasks", "GoJobs", "GoPlugins", "GoTasks", "NameSpaces":
			val = &tval
		case "LoadableModules":
			val = &mval
		case "ScheduledJobs":
			val = &stval
		case "DefaultChannels", "IgnoreUsers", "JoinChannels", "AdminUsers":
			val = &sarrval
		case "MailConfig":
			val = &mailval
		case "ProtocolConfig", "BrainConfig", "HistoryConfig":
			skip = true
		default:
			err := fmt.Errorf("Invalid configuration key in gopherbot.yaml: %s", key)
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
		case "LoadableModules":
			newconfig.LoadableModules = *(val.(*map[string]LoadableModule))
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

	if !preConnect {
		currentCfg.Lock()
	}
	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		if !strings.ContainsRune(string(aliases+escapeAliases), alias) {
			currentCfg.Unlock()
			return fmt.Errorf("Invalid alias specified, ignoring. Must be one of: %s%s", escapeAliases, aliases)
		}
		currentCfg.alias = alias
	}

	if len(newconfig.DefaultMessageFormat) == 0 {
		currentCfg.defaultMessageFormat = robot.Raw
	} else {
		currentCfg.defaultMessageFormat = setFormat(newconfig.DefaultMessageFormat)
	}

	if explicitDefaultAllowDirect {
		currentCfg.defaultAllowDirect = newconfig.DefaultAllowDirect
	} else {
		currentCfg.defaultAllowDirect = true // rare case of defaulting to true
	}

	if newconfig.AdminContact != "" {
		currentCfg.adminContact = newconfig.AdminContact
	}

	if newconfig.TimeZone != "" {
		tz, err := time.LoadLocation(newconfig.TimeZone)
		if err == nil {
			Log(robot.Info, "Set timezone: %s", tz)
			currentCfg.timeZone = tz
		} else {
			Log(robot.Error, "Parsing time zone '%s', using local time; error: %v", newconfig.TimeZone, err)
			currentCfg.timeZone = nil
		}
	}

	if newconfig.BotInfo != nil {
		botID := currentCfg.botinfo.UserID
		botMention := currentCfg.botinfo.protoMention
		currentCfg.botinfo = *newconfig.BotInfo
		currentCfg.botinfo.UserID = botID
		currentCfg.botinfo.protoMention = botMention
	}
	currentCfg.mailConf = newconfig.MailConfig

	if newconfig.DefaultJobChannel != "" {
		currentCfg.defaultJobChannel = newconfig.DefaultJobChannel
	}

	if newconfig.DefaultElevator != "" {
		currentCfg.defaultElevator = newconfig.DefaultElevator
	}

	if newconfig.DefaultAuthorizer != "" {
		currentCfg.defaultAuthorizer = newconfig.DefaultAuthorizer
	}

	if newconfig.AdminUsers != nil {
		currentCfg.adminUsers = newconfig.AdminUsers
	} else {
		currentCfg.adminUsers = []string{}
	}
	if newconfig.DefaultChannels != nil {
		currentCfg.plugChannels = newconfig.DefaultChannels
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
		currentCfg.externalPlugins = et
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
		currentCfg.externalJobs = et
	}
	if newconfig.ExternalTasks != nil {
		et := make([]TaskSettings, 0)
		for name, task := range newconfig.ExternalTasks {
			if task.Disabled {
				continue
			}
			task.Name = name
			et = append(et, task)
		}
		currentCfg.externalTasks = et
	}
	if newconfig.LoadableModules != nil {
		lm := make([]LoadableModule, 0)
		for name, mod := range newconfig.LoadableModules {
			mod.Name = name
			lm = append(lm, mod)
		}
		currentCfg.loadableModules = lm
	}
	st := make([]ScheduledTask, 0, len(newconfig.ScheduledJobs))
	for _, s := range newconfig.ScheduledJobs {
		if len(s.Name) == 0 || len(s.Schedule) == 0 {
			Log(robot.Error, "Zero-length Name (%s) or Schedule (%s) in ScheduledTask, skipping", s.Name, s.Schedule)
		} else {
			st = append(st, s)
		}
	}
	currentCfg.ScheduledJobs = st
	if newconfig.IgnoreUsers != nil {
		currentCfg.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		currentCfg.joinChannels = newconfig.JoinChannels
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
				Log(robot.Error, "one of Username/UserID empty (%s/%s), ignoring", user.UserName, user.UserID)
			} else {
				u := &newconfig.UserRoster[i]
				ucmaps.user[u.UserName] = u
				ucmaps.userID[u.UserID] = u
				usermap[u.UserName] = u.UserID
			}
		}
		if len(currentCfg.botinfo.UserName) > 0 && len(currentCfg.botinfo.UserID) > 0 {
			usermap[currentCfg.botinfo.UserName] = currentCfg.botinfo.UserID
		}
	}
	if len(newconfig.ChannelRoster) > 0 {
		for i, ch := range newconfig.ChannelRoster {
			if len(ch.ChannelName) == 0 || len(ch.ChannelID) == 0 {
				Log(robot.Error, "one of ChannelName/ChannelID empty (%s/%s), ignoring", ch.ChannelName, ch.ChannelID)
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

	if len(newconfig.WorkSpace) > 0 {
		h := handler{}
		if err := h.GetDirectory(newconfig.WorkSpace); err == nil {
			currentCfg.workSpace = newconfig.WorkSpace
			Log(robot.Debug, "Setting workspace directory to '%s'", currentCfg.workSpace)
		} else {
			Log(robot.Error, "Getting WorkSpace directory '%s', using '%s': %v", newconfig.WorkSpace, configPath, err)
		}
	}
	if len(currentCfg.workSpace) == 0 {
		currentCfg.workSpace = configPath
	}

	if newconfig.HistoryProvider != "" {
		currentCfg.historyProvider = newconfig.HistoryProvider
	}
	if newconfig.HistoryConfig != nil {
		historyConfig = newconfig.HistoryConfig
	}

	// Items only read at start-up, before multi-threaded
	if preConnect {
		if newconfig.Protocol != "" {
			currentCfg.protocol = newconfig.Protocol
		} else {
			return fmt.Errorf("Protocol not specified in gopherbot.yaml")
		}
		if newconfig.ProtocolConfig != nil {
			protocolConfig = newconfig.ProtocolConfig
		}

		if newconfig.EncryptBrain {
			encryptBrain = true
		}
		if newconfig.EncryptionKey != "" {
			currentCfg.encryptionKey = newconfig.EncryptionKey
			newconfig.EncryptionKey = "XXXXXX" // too short to be valid anyway
		}
		if newconfig.Brain != "" {
			currentCfg.brainProvider = newconfig.Brain
		}
		if newconfig.BrainConfig != nil {
			brainConfig = newconfig.BrainConfig
		}
		if newconfig.LocalPort != 0 {
			currentCfg.port = fmt.Sprintf("%d", newconfig.LocalPort)
		} else {
			currentCfg.port = "0"
		}
		if len(newconfig.HistoryProvider) > 0 {
			if hprovider, ok := historyProviders[newconfig.HistoryProvider]; !ok {
				Log(robot.Fatal, "No provider registered for history type: \"%s\"", currentCfg.historyProvider)
			} else {
				hp := hprovider(handler{})
				interfaces.history = hp
			}
		}
	} else {
		if len(usermap) > 0 {
			interfaces.SetUserMap(usermap)
		}
		// We should never dump the brain key
		newconfig.EncryptionKey = "XXXXXX"
		// loadTaskConfig does it's own locking
		currentCfg.Unlock()
	}

	confLock.Lock()
	config = newconfig
	repositories = repolist
	confLock.Unlock()

	if err := c.loadTaskConfig(preConnect); err != nil {
		return err
	}

	if !preConnect {
		updateRegexes()
		scheduleTasks()
	}

	return nil
}
