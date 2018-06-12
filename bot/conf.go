package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ghodss/yaml"
)

/* conf.go - methods and types for reading and storing json configuration */

var protocolConfig, brainConfig, historyConfig json.RawMessage

// botconf specifies 'bot configuration, and is read from $GOPHER_CONFIGDIR/conf/gopherbot.yaml
type botconf struct {
	AdminContact         string           // Contact info for whomever administers the robot
	Email                string           // From: address when the robot wants to send an email
	MailConfig           botMailer        // configuration for sending email
	Protocol             string           // Name of the connector protocol to use, e.g. "slack"
	ProtocolConfig       json.RawMessage  // Protocol-specific configuration, type for unmarshalling arbitrary config
	Brain                string           // Type of Brain to use
	BrainConfig          json.RawMessage  // Brain-specific configuration, type for unmarshalling arbitrary config
	HistoryProvider      string           // Name of provider to use for storing and retrieving job/plugin histories
	HistoryConfig        json.RawMessage  // History provider specific configuration
	DefaultElevator      string           // Elevator plugin to use by default for ElevatedCommands and ElevateImmediateCommands
	DefaultAuthorizer    string           // Authorizer plugin to use by default for AuthorizedCommands, or when AuthorizeAllCommands = true
	DefaultMessageFormat string           // How the robot should format outgoing messages unless told otherwise; default: Raw
	Name                 string           // Name of the 'bot, specify here if the protocol doesn't supply it (slack does)
	DefaultAllowDirect   bool             // Whether plugins are available in a DM by default
	DefaultChannels      []string         // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers          []string         // Users the 'bot never talks to - like other bots
	JoinChannels         []string         // Channels the 'bot should join when it logs in (not supported by all protocols)
	DefaultJobChannel    string           // Where job status is posted by default
	DefaultJobChannels   []string         // Where users can issue the 'run job <foo>' command
	TimeZone             string           // For evaluating the hour in a job schedule
	Jobs                 []externalScript // list of available jobs; config in conf/jobs/<jobname.yaml>
	ScheduledTasks       []scheduledTask  // see tasks.go
	ExternalScripts      []externalScript // List of non-Go plugins to load
	AdminUsers           []string         // List of users who can access administrative commands
	Alias                string           // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort            int              // Port number for listening on localhost, for CLI plugins
	LogLevel             string           // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
}

var config *botconf

// getConfigFile loads a config file first from installPath, then from configPath
// if set.

// Required indicates whether to return an error if neither file is found.
func (r *botContext) getConfigFile(filename, callerID string, required bool, jsonMap map[string]json.RawMessage) error {
	var (
		cf           []byte
		err, realerr error
	)

	loaded := false
	var loader map[string]json.RawMessage
	var path string
	robot.RLock()
	installPath := robot.installPath
	configPath := robot.configPath
	robot.RUnlock()

	loader = make(map[string]json.RawMessage)
	path = installPath + "/conf/" + filename
	cf, err = ioutil.ReadFile(path)
	if err == nil {
		r.debug(fmt.Sprintf("Loaded configuration from installPath (%s), size: %d", path, len(cf)), false)
		if err = yaml.Unmarshal(cf, &loader); err != nil {
			err = fmt.Errorf("Unmarshalling installed \"%s\": %v", filename, err)
			Log(Error, err)
			return err
		}
		if len(loader) == 0 {
			msg := fmt.Sprintf("Empty config hash loading %s", path)
			r.debug(msg, false)
			Log(Error, msg)
		} else {
			for key, value := range loader {
				jsonMap[key] = value
			}
			Log(Debug, fmt.Sprintf("Loaded installed conf/%s", filename))
			loaded = true
		}
	} else {
		r.debug(fmt.Sprintf("No configuration loaded from installPath (%s): %v", path, err), false)
		realerr = err
	}
	if len(configPath) > 0 {
		loader = make(map[string]json.RawMessage)
		path = configPath + "/conf/" + filename
		cf, err = ioutil.ReadFile(path)
		if err == nil {
			r.debug(fmt.Sprintf("Loaded configuration from configPath (%s), size: %d", path, len(cf)), false)
			if err = yaml.Unmarshal(cf, &loader); err != nil {
				err = fmt.Errorf("Unmarshalling configured \"%s\": %v", filename, err)
				Log(Error, err)
				return err // If a badly-formatted config is loaded, we always return an error
			}
			if len(loader) == 0 {
				msg := fmt.Sprintf("Empty config hash loading %s", path)
				r.debug(msg, false)
				Log(Error, msg)
			} else {
				for key, value := range loader {
					jsonMap[key] = value
				}
				Log(Debug, fmt.Sprintf("Loaded configured conf/%s", filename))
				loaded = true
			}
		} else {
			r.debug(fmt.Sprintf("No configuration loaded from configPath (%s): %v", path, err), false)
			realerr = err
		}
	}
	if required && !loaded {
		return realerr
	}
	return nil
}

// loadConfig loads the 'bot's json configuration files.
func (r *botContext) loadConfig() error {
	var loglevel LogLevel
	newconfig := &botconf{}
	configload := make(map[string]json.RawMessage)
	pluginsOk := true

	if err := r.getConfigFile("gopherbot.yaml", "", true, configload); err != nil {
		return fmt.Errorf("Loading configuration file: %v", err)
	}
	explicitDefaultAllowDirect := false

	for key, value := range configload {
		var strval string
		var sarrval []string
		var epval, jval []externalScript
		var stval []scheduledTask
		var mailval botMailer
		var boolval bool
		var intval int
		var val interface{}
		skip := false
		switch key {
		case "AdminContact", "Email", "Protocol", "Brain", "HistoryProvider", "DefaultJobChannel", "DefaultElevator", "DefaultAuthorizer", "DefaultMessageFormat", "Name", "Alias", "LogLevel", "TimeZone":
			val = &strval
		case "DefaultAllowDirect":
			val = &boolval
		case "LocalPort":
			val = &intval
		case "ExternalScripts":
			val = &epval
		case "Jobs":
			val = &jval
		case "ScheduledJobs":
			val = &stval
		case "DefaultChannels", "DefaultJobChannels", "IgnoreUsers", "JoinChannels", "AdminUsers":
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
		case "BrainConfig":
			newconfig.BrainConfig = value
		case "HistoryProvider":
			newconfig.HistoryProvider = *(val.(*string))
		case "HistoryConfig":
			newconfig.HistoryConfig = value
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
		case "DefaultJobChannels":
			newconfig.DefaultJobChannels = *(val.(*[]string))
		case "IgnoreUsers":
			newconfig.IgnoreUsers = *(val.(*[]string))
		case "JoinChannels":
			newconfig.JoinChannels = *(val.(*[]string))
		case "ExternalScripts":
			newconfig.ExternalScripts = *(val.(*[]externalScript))
		case "Jobs":
			newconfig.Jobs = *(val.(*[]externalScript))
		case "ScheduledJobs":
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

	robot.Lock()
	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		if !strings.ContainsRune(string(aliases+escapeAliases), alias) {
			robot.Unlock()
			return fmt.Errorf("Invalid alias specified, ignoring. Must be one of: %s%s", escapeAliases, aliases)
		}
		robot.alias = alias
	}
	if newconfig.Protocol != "" {
		robot.protocol = newconfig.Protocol
	} else {
		robot.Unlock()
		return fmt.Errorf("Protocol not specified in gopherbot.yaml")
	}

	if len(newconfig.DefaultMessageFormat) == 0 {
		robot.defaultMessageFormat = Raw
	} else {
		robot.defaultMessageFormat = r.makeRobot().setFormat(newconfig.DefaultMessageFormat)
	}

	if newconfig.ProtocolConfig != nil {
		protocolConfig = newconfig.ProtocolConfig
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
		if err != nil {
			robot.timeZone = tz
		} else {
			Log(Error, fmt.Errorf("Parsing time zone '%s', using local time; error: %q", newconfig.TimeZone, err))
			robot.timeZone = nil
		}
	}

	if newconfig.Email != "" {
		robot.email = newconfig.Email
	}
	robot.mailConf = newconfig.MailConfig
	if newconfig.LocalPort != 0 {
		robot.port = fmt.Sprintf("127.0.0.1:%d", newconfig.LocalPort)
		err := os.Setenv("GOPHER_HTTP_POST", "http://"+robot.port)
		if err != nil {
			Log(Error, fmt.Errorf("Error exporting GOPHER_HTTP_PORT: %q", err))
		}
	} else {
		Log(Error, "LocalPort not defined, not exporting GOPHER_HTTP_POST and external plugins will be broken")
	}
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

	if newconfig.Brain != "" {
		robot.brainProvider = newconfig.Brain
	}
	if newconfig.BrainConfig != nil {
		brainConfig = newconfig.BrainConfig
	}

	if newconfig.HistoryProvider != "" {
		robot.historyProvider = newconfig.HistoryProvider
	}
	if newconfig.HistoryConfig != nil {
		historyConfig = newconfig.HistoryConfig
	}

	if newconfig.AdminUsers != nil {
		robot.adminUsers = newconfig.AdminUsers
	}
	if newconfig.DefaultChannels != nil {
		robot.plugChannels = newconfig.DefaultChannels
	}
	if newconfig.ExternalScripts != nil {
		for i, ep := range newconfig.ExternalScripts {
			if len(ep.Name) == 0 || len(ep.Path) == 0 {
				pluginsOk = false
				Log(Error, fmt.Errorf("Reading external plugins, zero-length Name or Path for plugin #%d, not reloading plugins", i))
			}
		}
		if pluginsOk {
			robot.externalScripts = newconfig.ExternalScripts
		}
	}
	if newconfig.IgnoreUsers != nil {
		robot.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		robot.joinChannels = newconfig.JoinChannels
	}

	// loadTaskConfig does it's own locking
	robot.Unlock()

	globalLock.Lock()
	config = newconfig
	globalLock.Unlock()

	updateRegexes()
	if pluginsOk {
		r.loadTaskConfig()
	} else {
		return fmt.Errorf("Error reading external plugin config")
	}

	return nil
}
