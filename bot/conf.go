package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
)

/* conf.go - methods and types for reading and storing json configuration */

var protocolConfig, brainConfig, historyConfig json.RawMessage

// BotConf defines 'bot configuration, and is read from conf/gopherbot.yaml
type BotConf struct {
	AdminContact         string                  // Contact info for whomever administers the robot
	Email                string                  // From: address when the robot wants to send an email
	MailConfig           botMailer               // configuration for sending email
	Protocol             string                  // Name of the connector protocol to use, e.g. "slack"
	ProtocolConfig       json.RawMessage         // Protocol-specific configuration, type for unmarshalling arbitrary config
	Brain                string                  // Type of Brain to use
	BrainConfig          json.RawMessage         // Brain-specific configuration, type for unmarshalling arbitrary config
	EncryptBrain         bool                    // Whether the brain should be encrypted
	EncryptionKey        string                  // used to decrypt the "real" encryption key
	HistoryProvider      string                  // Name of provider to use for storing and retrieving job/plugin histories
	HistoryConfig        json.RawMessage         // History provider specific configuration
	WorkSpace            string                  // Read/Write area the robot uses to do work
	DefaultElevator      string                  // Elevator plugin to use by default for ElevatedCommands and ElevateImmediateCommands
	DefaultAuthorizer    string                  // Authorizer plugin to use by default for AuthorizedCommands, or when AuthorizeAllCommands = true
	DefaultMessageFormat string                  // How the robot should format outgoing messages unless told otherwise; default: Raw
	Name                 string                  // Name of the 'bot, specify here if the protocol doesn't supply it (slack does)
	DefaultAllowDirect   bool                    // Whether plugins are available in a DM by default
	DefaultChannels      []string                // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers          []string                // Users the 'bot never talks to - like other bots
	JoinChannels         []string                // Channels the 'bot should join when it logs in (not supported by all protocols)
	DefaultJobChannel    string                  // Where job status is posted by default
	TimeZone             string                  // For evaluating the hour in a job schedule
	ExternalJobs         map[string]ExternalTask // list of available jobs; config in conf/jobs/<jobname>.yaml
	ExternalPlugins      map[string]ExternalTask // List of non-Go plugins to load; config in conf/plugins/<plugname>.yaml
	ExternalTasks        map[string]ExternalTask // List executables that can be added to a pipeline (but can't start one)
	ScheduledTasks       []ScheduledTask         // see tasks.go
	AdminUsers           []string                // List of users who can access administrative commands
	Alias                string                  // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort            int                     // Port number for listening on localhost, for CLI plugins
	LogLevel             string                  // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
}

type repository struct {
	Parameters []Parameter // per-repository parameters
}

// Protects the bot config and list of repositories
var confLock sync.RWMutex
var config *BotConf
var repositories map[string]repository

// loadConfig loads the 'bot's yaml configuration files.
func (c *botContext) loadConfig(preConnect bool) error {
	if err := godotenv.Overload("gopherbot.env"); err == nil {
		Log(Info, "Loaded environment from 'gopherbot.env'")
	}
	var loglevel LogLevel
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
			Log(Error, fmt.Sprintf("Invalid repository '%s' contains ':', ignoring", k))
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
		var tval map[string]ExternalTask
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
		case "LocalPort":
			val = &intval
		case "ExternalJobs", "ExternalPlugins", "ExternalTasks":
			val = &tval
		case "ScheduledTasks":
			val = &stval
		case "DefaultChannels", "IgnoreUsers", "JoinChannels", "AdminUsers":
			val = &sarrval
		case "MailConfig":
			val = &mailval
		case "ProtocolConfig", "BrainConfig", "HistoryConfig":
			skip = true
		default:
			err := fmt.Errorf("Invalid configuration key in gopherbot.yaml: %s", key)
			Log(Error, err)
			return err
		}
		if !skip {
			if err := json.Unmarshal(value, val); err != nil {
				err = fmt.Errorf("Unmarshalling bot config value \"%s\": %v", key, err)
				Log(Error, err)
				return err
			}
		}
		switch key {
		case "AdminContact":
			newconfig.AdminContact = *(val.(*string))
		case "Email":
			newconfig.Email = *(val.(*string))
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
		case "Name":
			newconfig.Name = *(val.(*string))
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
		case "ScheduledTasks":
			newconfig.ScheduledTasks = *(val.(*[]ScheduledTask))
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

	r := c.makeRobot()
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
		botCfg.defaultMessageFormat = Raw
	} else {
		botCfg.defaultMessageFormat = r.setFormat(newconfig.DefaultMessageFormat)
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
			Log(Info, fmt.Sprintf("Set timezone: %s", tz))
			botCfg.timeZone = tz
		} else {
			Log(Error, fmt.Errorf("Parsing time zone '%s', using local time; error: %v", newconfig.TimeZone, err))
			botCfg.timeZone = nil
		}
	}

	if newconfig.Email != "" {
		botCfg.email = newconfig.Email
	}
	botCfg.mailConf = newconfig.MailConfig

	if newconfig.Name != "" {
		botCfg.name = newconfig.Name
	}

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
	st := make([]ScheduledTask, 0, len(newconfig.ScheduledTasks))
	for _, s := range newconfig.ScheduledTasks {
		if len(s.Name) == 0 || len(s.Schedule) == 0 {
			Log(Error, fmt.Sprintf("Zero-length Name (%s) or Schedule (%s) in ScheduledTask, skipping", s.Name, s.Schedule))
		} else {
			st = append(st, s)
		}
	}
	botCfg.ScheduledTasks = st
	if newconfig.IgnoreUsers != nil {
		botCfg.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		botCfg.joinChannels = newconfig.JoinChannels
	}

	if len(newconfig.WorkSpace) > 0 {
		if respath, ok := checkDirectory(newconfig.WorkSpace); ok {
			botCfg.workSpace = respath
		} else {
			Log(Error, fmt.Sprintf("WorkSpace directory '%s' doesn't exist, using '%s'", newconfig.WorkSpace, configPath))
		}
	}
	if len(botCfg.workSpace) == 0 {
		botCfg.workSpace = configPath
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
			Log(Error, "LocalPort not defined, not exporting GOPHER_HTTP_POST and external tasks will be broken")
		}
		if newconfig.HistoryProvider != "" {
			botCfg.historyProvider = newconfig.HistoryProvider
		}
		if newconfig.HistoryConfig != nil {
			historyConfig = newconfig.HistoryConfig
		}
	} else {
		// We should never dump the brain key
		newconfig.EncryptionKey = "XXXXXX"
		// loadTaskConfig does it's own locking
		botCfg.Unlock()
	}

	confLock.Lock()
	config = newconfig
	repositories = repolist
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
