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

// BotConf defines 'bot configuration, and is read from conf/gopherbot.yaml
type BotConf struct {
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
	ExternalJobs         map[string]ExternalTask   // list of available jobs; config in conf/jobs/<jobname>.yaml
	ExternalPlugins      map[string]ExternalTask   // List of non-Go plugins to load; config in conf/plugins/<plugname>.yaml
	ExternalTasks        map[string]ExternalTask   // List executables that can be added to a pipeline (but can't start one)
	LoadableModules      map[string]LoadableModule // List of loadable modules to load
	ScheduledJobs        []ScheduledTask           // see tasks.go
	AdminUsers           []string                  // List of users who can access administrative commands
	Alias                string                    // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort            int                       // Port number for listening on localhost, for CLI plugins
	LogLevel             string                    // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
}

type repository struct {
	Parameters []Parameter // per-repository parameters
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
var config *BotConf
var repositories map[string]repository
var repodata map[string]json.RawMessage

// loadConfig loads the 'bot's yaml configuration files.
func (c *botContext) loadConfig(preConnect bool) error {
	var loglevel robot.LogLevel
	newconfig := &BotConf{}
	newconfig.ExternalJobs = make(map[string]ExternalTask)
	newconfig.ExternalPlugins = make(map[string]ExternalTask)
	newconfig.ExternalTasks = make(map[string]ExternalTask)
	configload := make(map[string]json.RawMessage)
	tasksOk := true

	if err := c.getConfigFile("gopherbot.yaml", "", true, configload); err != nil {
		return fmt.Errorf("Loading configuration file: %v", err)
	}

	reporaw := make(map[string]json.RawMessage)
	c.getConfigFile("repositories.yaml", "", false, reporaw)
	repolist := make(map[string]repository)
	for k, repojson := range reporaw {
		if strings.ContainsRune(k, ':') {
			Log(robot.Error, "Invalid repository '%s' contains ':', ignoring", k)
		} else {
			var repository repository
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
		var tval map[string]ExternalTask
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
		case "ExternalJobs", "ExternalPlugins", "ExternalTasks":
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
			newconfig.ExternalPlugins = *(val.(*map[string]ExternalTask))
		case "ExternalJobs":
			newconfig.ExternalJobs = *(val.(*map[string]ExternalTask))
		case "ExternalTasks":
			newconfig.ExternalTasks = *(val.(*map[string]ExternalTask))
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

	loglevel = logStrToLevel(newconfig.LogLevel)
	setLogLevel(loglevel)

	if !preConnect {
		botCfg.Lock()
	}
	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		if !strings.ContainsRune(string(aliases+escapeAliases), alias) {
			botCfg.Unlock()
			return fmt.Errorf("Invalid alias specified, ignoring. Must be one of: %s%s", escapeAliases, aliases)
		}
		botCfg.alias = alias
	}

	if len(newconfig.DefaultMessageFormat) == 0 {
		botCfg.defaultMessageFormat = robot.Raw
	} else {
		botCfg.defaultMessageFormat = setFormat(newconfig.DefaultMessageFormat)
	}

	if explicitDefaultAllowDirect {
		botCfg.defaultAllowDirect = newconfig.DefaultAllowDirect
	} else {
		botCfg.defaultAllowDirect = true // rare case of defaulting to true
	}

	if newconfig.AdminContact != "" {
		botCfg.adminContact = newconfig.AdminContact
	}

	if newconfig.TimeZone != "" {
		tz, err := time.LoadLocation(newconfig.TimeZone)
		if err == nil {
			Log(robot.Info, "Set timezone: %s", tz)
			botCfg.timeZone = tz
		} else {
			Log(robot.Error, "Parsing time zone '%s', using local time; error: %v", newconfig.TimeZone, err)
			botCfg.timeZone = nil
		}
	}

	if newconfig.BotInfo != nil {
		botID := botCfg.botinfo.UserID
		botMention := botCfg.botinfo.protoMention
		botCfg.botinfo = *newconfig.BotInfo
		botCfg.botinfo.UserID = botID
		botCfg.botinfo.protoMention = botMention
	}
	botCfg.mailConf = newconfig.MailConfig

	if newconfig.DefaultJobChannel != "" {
		botCfg.defaultJobChannel = newconfig.DefaultJobChannel
	}

	if newconfig.DefaultElevator != "" {
		botCfg.defaultElevator = newconfig.DefaultElevator
	}

	if newconfig.DefaultAuthorizer != "" {
		botCfg.defaultAuthorizer = newconfig.DefaultAuthorizer
	}

	if newconfig.AdminUsers != nil {
		botCfg.adminUsers = newconfig.AdminUsers
	} else {
		botCfg.adminUsers = []string{}
	}
	if newconfig.DefaultChannels != nil {
		botCfg.plugChannels = newconfig.DefaultChannels
	}
	// TODO: skip when disabled, use append instead of making array to length.
	if newconfig.ExternalPlugins != nil {
		ni := len(newconfig.ExternalPlugins)
		et := make([]ExternalTask, ni)
		i := 0
		for name, task := range newconfig.ExternalPlugins {
			et[i] = task
			et[i].Name = name
			i++
		}
		botCfg.externalPlugins = et
	}
	if newconfig.ExternalJobs != nil {
		ni := len(newconfig.ExternalJobs)
		et := make([]ExternalTask, ni)
		i := 0
		for name, task := range newconfig.ExternalJobs {
			et[i] = task
			et[i].Name = name
			i++
		}
		botCfg.externalJobs = et
	}
	if newconfig.ExternalTasks != nil {
		ni := len(newconfig.ExternalTasks)
		et := make([]ExternalTask, ni)
		i := 0
		for name, task := range newconfig.ExternalTasks {
			et[i] = task
			et[i].Name = name
			i++
		}
		botCfg.externalTasks = et
	}
	if newconfig.LoadableModules != nil {
		ni := len(newconfig.LoadableModules)
		lm := make([]LoadableModule, ni)
		i := 0
		for name, mod := range newconfig.LoadableModules {
			lm[i] = mod
			lm[i].Name = name
			i++
		}
		botCfg.loadableModules = lm
	}
	st := make([]ScheduledTask, 0, len(newconfig.ScheduledJobs))
	for _, s := range newconfig.ScheduledJobs {
		if len(s.Name) == 0 || len(s.Schedule) == 0 {
			Log(robot.Error, "Zero-length Name (%s) or Schedule (%s) in ScheduledTask, skipping", s.Name, s.Schedule)
		} else {
			st = append(st, s)
		}
	}
	botCfg.ScheduledJobs = st
	if newconfig.IgnoreUsers != nil {
		botCfg.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		botCfg.joinChannels = newconfig.JoinChannels
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
		if len(botCfg.botinfo.UserName) > 0 && len(botCfg.botinfo.UserID) > 0 {
			usermap[botCfg.botinfo.UserName] = botCfg.botinfo.UserID
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
		if respath, ok := checkDirectory(newconfig.WorkSpace); ok {
			botCfg.workSpace = respath
			Log(robot.Debug, "Setting workspace directory to '%s'", respath)
		} else {
			Log(robot.Error, "WorkSpace directory '%s' doesn't exist, using '%s'", newconfig.WorkSpace, configPath)
		}
	}
	if len(botCfg.workSpace) == 0 {
		botCfg.workSpace = configPath
	}

	if newconfig.HistoryProvider != "" {
		botCfg.historyProvider = newconfig.HistoryProvider
	}
	if newconfig.HistoryConfig != nil {
		historyConfig = newconfig.HistoryConfig
	}

	// Items only read at start-up, before multi-threaded
	if preConnect {
		if newconfig.Protocol != "" {
			botCfg.protocol = newconfig.Protocol
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
			botCfg.encryptionKey = newconfig.EncryptionKey
			newconfig.EncryptionKey = "XXXXXX" // too short to be valid anyway
		}
		if newconfig.Brain != "" {
			botCfg.brainProvider = newconfig.Brain
		}
		if newconfig.BrainConfig != nil {
			brainConfig = newconfig.BrainConfig
		}
		if newconfig.LocalPort != 0 {
			botCfg.port = fmt.Sprintf("127.0.0.1:%d", newconfig.LocalPort)
		} else {
			Log(robot.Error, "LocalPort not defined, not exporting GOPHER_HTTP_POST and external tasks will be broken")
		}
	} else {
		if len(usermap) > 0 {
			botCfg.SetUserMap(usermap)
		}
		// We should never dump the brain key
		newconfig.EncryptionKey = "XXXXXX"
		// loadTaskConfig does it's own locking
		historyConfigured := botCfg.history != nil
		botCfg.Unlock()
		if !historyConfigured && len(newconfig.HistoryProvider) > 0 {
			if hprovider, ok := historyProviders[newconfig.HistoryProvider]; !ok {
				Log(robot.Fatal, "No provider registered for history type: \"%s\"", botCfg.historyProvider)
			} else {
				hp := hprovider(handler{})
				botCfg.Lock()
				botCfg.history = hp
				botCfg.Unlock()
			}
		}
	}

	confLock.Lock()
	config = newconfig
	repositories = repolist
	repodata = reporaw
	confLock.Unlock()

	if tasksOk && !preConnect {
		c.loadTaskConfig()
	} else if !tasksOk {
		return fmt.Errorf("Error reading external plugin config")
	}

	if !preConnect {
		updateRegexes()
		scheduleTasks()
	}

	return nil
}
