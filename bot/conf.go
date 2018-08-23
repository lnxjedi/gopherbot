package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ghodss/yaml"
)

/* conf.go - methods and types for reading and storing json configuration */

var protocolConfig, brainConfig, historyConfig json.RawMessage

// botconf specifies 'bot configuration, and is read from $GOPHER_CONFIGDIR/conf/gopherbot.yaml
type botconf struct {
	AdminContact         string          // Contact info for whomever administers the robot
	Email                string          // From: address when the robot wants to send an email
	MailConfig           botMailer       // configuration for sending email
	Protocol             string          // Name of the connector protocol to use, e.g. "slack"
	ProtocolConfig       json.RawMessage // Protocol-specific configuration, type for unmarshalling arbitrary config
	Brain                string          // Type of Brain to use
	BrainConfig          json.RawMessage // Brain-specific configuration, type for unmarshalling arbitrary config
	EncryptBrain         bool            // Whether the brain should be encrypted
	BrainKey             string          // used to decrypt the brainKey
	HistoryProvider      string          // Name of provider to use for storing and retrieving job/plugin histories
	HistoryConfig        json.RawMessage // History provider specific configuration
	WorkSpace            string          // Read/Write area the robot uses to do work
	DefaultElevator      string          // Elevator plugin to use by default for ElevatedCommands and ElevateImmediateCommands
	DefaultAuthorizer    string          // Authorizer plugin to use by default for AuthorizedCommands, or when AuthorizeAllCommands = true
	DefaultMessageFormat string          // How the robot should format outgoing messages unless told otherwise; default: Raw
	Name                 string          // Name of the 'bot, specify here if the protocol doesn't supply it (slack does)
	DefaultAllowDirect   bool            // Whether plugins are available in a DM by default
	DefaultChannels      []string        // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers          []string        // Users the 'bot never talks to - like other bots
	JoinChannels         []string        // Channels the 'bot should join when it logs in (not supported by all protocols)
	DefaultJobChannel    string          // Where job status is posted by default
	TimeZone             string          // For evaluating the hour in a job schedule
	ExternalJobs         []externalTask  // list of available jobs; config in conf/jobs/<jobname>.yaml
	ScheduledTasks       []scheduledTask // see tasks.go
	ExternalPlugins      []externalTask  // List of non-Go plugins to load; config in conf/plugins/<plugname>.yaml
	ExternalTasks        []externalTask  // List executables that can be added to a pipeline (but can't start one)
	AdminUsers           []string        // List of users who can access administrative commands
	Alias                string          // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort            int             // Port number for listening on localhost, for CLI plugins
	LogLevel             string          // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
}

type repository struct {
	Parameters []parameter // per-repository parameters
}

// Protects the bot config and list of repositories
var confLock sync.RWMutex
var config *botconf
var repositories map[string]repository

// getConfigFile loads a config file first from installPath, then from configPath
// if set.

// Required indicates whether to return an error if neither file is found.
func (c *botContext) getConfigFile(filename, callerID string, required bool, jsonMap map[string]json.RawMessage) error {
	var (
		cf           []byte
		err, realerr error
	)

	loaded := false
	var loader map[string]json.RawMessage
	var path string

	loader = make(map[string]json.RawMessage)
	path = installPath + "/conf/" + filename
	cf, err = ioutil.ReadFile(path)
	if err == nil {
		c.debug(fmt.Sprintf("Loaded configuration from installPath (%s), size: %d", path, len(cf)), false)
		if err = yaml.Unmarshal(cf, &loader); err != nil {
			err = fmt.Errorf("Unmarshalling installed \"%s\": %v", filename, err)
			Log(Error, err)
			return err
		}
		if len(loader) == 0 {
			msg := fmt.Sprintf("Empty config hash loading %s", path)
			c.debug(msg, false)
			Log(Error, msg)
		} else {
			for key, value := range loader {
				jsonMap[key] = value
			}
			Log(Debug, fmt.Sprintf("Loaded installed conf/%s", filename))
			loaded = true
		}
	} else {
		c.debug(fmt.Sprintf("No configuration loaded from installPath (%s): %v", path, err), false)
		realerr = err
	}
	if len(configPath) > 0 {
		loader = make(map[string]json.RawMessage)
		path = configPath + "/conf/" + filename
		cf, err = ioutil.ReadFile(path)
		if err == nil {
			c.debug(fmt.Sprintf("Loaded configuration from configPath (%s), size: %d", path, len(cf)), false)
			if err = yaml.Unmarshal(cf, &loader); err != nil {
				err = fmt.Errorf("Unmarshalling configured \"%s\": %v", filename, err)
				Log(Error, err)
				return err // If a badly-formatted config is loaded, we always return an error
			}
			if len(loader) == 0 {
				msg := fmt.Sprintf("Empty config hash loading %s", path)
				c.debug(msg, false)
				Log(Error, msg)
			} else {
				for key, value := range loader {
					jsonMap[key] = value
				}
				Log(Debug, fmt.Sprintf("Loaded configured conf/%s", filename))
				loaded = true
			}
		} else {
			c.debug(fmt.Sprintf("No configuration loaded from configPath (%s): %v", path, err), false)
			realerr = err
		}
	}
	if required && !loaded {
		return realerr
	}
	return nil
}

// loadConfig loads the 'bot's json configuration files.
func (c *botContext) loadConfig(preConnect bool) error {
	var loglevel LogLevel
	newconfig := &botconf{}
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
		var tval []externalTask
		var stval []scheduledTask
		var mailval botMailer
		var boolval bool
		var intval int
		var val interface{}
		skip := false
		switch key {
		case "AdminContact", "Email", "Protocol", "Brain", "BrainKey", "HistoryProvider", "WorkSpace", "DefaultJobChannel", "DefaultElevator", "DefaultAuthorizer", "DefaultMessageFormat", "Name", "Alias", "LogLevel", "TimeZone":
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
		case "BrainKey":
			newconfig.BrainKey = *(val.(*string))
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
			newconfig.ExternalPlugins = *(val.(*[]externalTask))
		case "ExternalJobs":
			newconfig.ExternalJobs = *(val.(*[]externalTask))
		case "ExternalTasks":
			newconfig.ExternalTasks = *(val.(*[]externalTask))
		case "ScheduledTasks":
			newconfig.ScheduledTasks = *(val.(*[]scheduledTask))
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
		robot.Lock()
	}
	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		if !strings.ContainsRune(string(aliases+escapeAliases), alias) {
			robot.Unlock()
			return fmt.Errorf("Invalid alias specified, ignoring. Must be one of: %s%s", escapeAliases, aliases)
		}
		robot.alias = alias
	}

	if len(newconfig.DefaultMessageFormat) == 0 {
		robot.defaultMessageFormat = Raw
	} else {
		robot.defaultMessageFormat = r.setFormat(newconfig.DefaultMessageFormat)
	}

	if explicitDefaultAllowDirect {
		robot.defaultAllowDirect = newconfig.DefaultAllowDirect
	} else {
		robot.defaultAllowDirect = true // rare case of defaulting to true
	}

	if newconfig.AdminContact != "" {
		robot.adminContact = newconfig.AdminContact
	}

	if newconfig.TimeZone != "" {
		tz, err := time.LoadLocation(newconfig.TimeZone)
		if err == nil {
			Log(Info, fmt.Sprintf("Set timezone: %s", tz))
			robot.timeZone = tz
		} else {
			Log(Error, fmt.Errorf("Parsing time zone '%s', using local time; error: %v", newconfig.TimeZone, err))
			robot.timeZone = nil
		}
	}

	if newconfig.Email != "" {
		robot.email = newconfig.Email
	}
	robot.mailConf = newconfig.MailConfig

	if newconfig.Name != "" {
		robot.name = newconfig.Name
	}

	if newconfig.DefaultJobChannel != "" {
		robot.defaultJobChannel = newconfig.DefaultJobChannel
	}

	if newconfig.DefaultElevator != "" {
		robot.defaultElevator = newconfig.DefaultElevator
	}

	if newconfig.DefaultAuthorizer != "" {
		robot.defaultAuthorizer = newconfig.DefaultAuthorizer
	}

	if newconfig.AdminUsers != nil {
		robot.adminUsers = newconfig.AdminUsers
	}
	if newconfig.DefaultChannels != nil {
		robot.plugChannels = newconfig.DefaultChannels
	}
	if newconfig.ExternalPlugins != nil {
		for i, ep := range newconfig.ExternalPlugins {
			if len(ep.Name) == 0 {
				tasksOk = false
				Log(Error, fmt.Errorf("Reading external plugins, zero-length Name for plugin #%d, not reloading plugins", i))
			}
		}
		if tasksOk {
			robot.externalPlugins = newconfig.ExternalPlugins
		}
	}
	if newconfig.ExternalJobs != nil {
		for i, ep := range newconfig.ExternalJobs {
			if len(ep.Name) == 0 {
				tasksOk = false
				Log(Error, fmt.Errorf("Reading external jobs, zero-length Name for job #%d, not reloading jobs", i))
			}
		}
		if tasksOk {
			robot.externalJobs = newconfig.ExternalJobs
		}
	}
	if newconfig.ExternalTasks != nil {
		for i, et := range newconfig.ExternalTasks {
			if len(et.Name) == 0 || len(et.Path) == 0 {
				tasksOk = false
				Log(Error, fmt.Errorf("Reading external tasks, zero-length Name or Path for task #%d, not reloading tasks", i))
			}
		}
		if tasksOk {
			robot.externalTasks = newconfig.ExternalTasks
		}
	}
	st := make([]scheduledTask, 0, len(newconfig.ScheduledTasks))
	for _, s := range newconfig.ScheduledTasks {
		if len(s.Name) == 0 || len(s.Schedule) == 0 {
			Log(Error, fmt.Sprintf("Zero-length Name (%s) or Schedule (%s) in ScheduledTask, skipping", s.Name, s.Schedule))
		} else {
			st = append(st, s)
		}
	}
	robot.scheduledTasks = st
	if newconfig.IgnoreUsers != nil {
		robot.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		robot.joinChannels = newconfig.JoinChannels
	}

	// Items only read at start-up, before multi-threaded
	if preConnect {
		if newconfig.Protocol != "" {
			robot.protocol = newconfig.Protocol
		} else {
			return fmt.Errorf("Protocol not specified in gopherbot.yaml")
		}
		if newconfig.ProtocolConfig != nil {
			protocolConfig = newconfig.ProtocolConfig
		}

		var p string
		if len(configPath) > 0 {
			p = configPath
		} else {
			p = installPath
		}
		if newconfig.WorkSpace != "" {
			if dirExists(newconfig.WorkSpace) {
				robot.workSpace = newconfig.WorkSpace
			} else {
				Log(Error, fmt.Sprintf("WorkSpace directory '%s' doesn't exist, using '%s'", newconfig.WorkSpace, p))
			}
		}
		if len(robot.workSpace) == 0 {
			robot.workSpace = p
		}

		if newconfig.EncryptBrain {
			encryptBrain = true
		}
		if newconfig.BrainKey != "" {
			robot.brainKey = newconfig.BrainKey
		}
		if newconfig.Brain != "" {
			robot.brainProvider = newconfig.Brain
		}
		if newconfig.BrainConfig != nil {
			brainConfig = newconfig.BrainConfig
		}
		if newconfig.LocalPort != 0 {
			robot.port = fmt.Sprintf("127.0.0.1:%d", newconfig.LocalPort)
		} else {
			Log(Error, "LocalPort not defined, not exporting GOPHER_HTTP_POST and external tasks will be broken")
		}
		if newconfig.HistoryProvider != "" {
			robot.historyProvider = newconfig.HistoryProvider
		}
		if newconfig.HistoryConfig != nil {
			historyConfig = newconfig.HistoryConfig
		}
	} else {
		// We should never dump the brain key
		newconfig.BrainKey = "XXXXXX"
		// loadTaskConfig does it's own locking
		robot.Unlock()
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
