package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ghodss/yaml"
)

/* conf.go - methods and types for reading and storing json configuration */

var protocolConfig, brainConfig, elevateConfig json.RawMessage

type externalPlugin struct {
	Name, Path string // List of names and paths for external plugins; relative paths are searched first in installdir, then localdir
}

// botconf specifies 'bot configuration, and is read from $GOPHER_CONFIGDIR/conf/gopherbot.yaml
type botconf struct {
	AdminContact       string           // Contact info for whomever administers the robot
	Email              string           // From: address when the robot wants to send an email
	MailConfig         botMailer        // configuration for sending email
	Protocol           string           // Name of the connector protocol to use, e.g. "slack"
	ProtocolConfig     json.RawMessage  // Protocol-specific configuration, type for unmarshalling arbitrary config
	Brain              string           // Type of Brain to use
	BrainConfig        json.RawMessage  // Brain-specific configuration, type for unmarshalling arbitrary config
	DefaultElevator    string           // Elevator plugin to use by default for ElevatedCommands and ElevateImmediateCommands
	DefaultAuthorizer  string           // Authorizer plugin to use by default for AuthorizedCommands, or when AuthorizeAllCommands = true
	Name               string           // Name of the 'bot, specify here if the protocol doesn't supply it (slack does)
	DefaultAllowDirect bool             // Whether plugins are available in a DM by default
	DefaultChannels    []string         // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers        []string         // Users the 'bot never talks to - like other bots
	JoinChannels       []string         // Channels the 'bot should join when it logs in (not supported by all protocols)
	ExternalPlugins    []externalPlugin // List of non-Go plugins to load
	AdminUsers         []string         // List of users who can access administrative commands
	Alias              string           // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort          int              // Port number for listening on localhost, for CLI plugins
	LogLevel           string           // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
}

var config *botconf

// getConfigFile loads a config file first from installPath, then from localPath.
// The goal is to support prod/dev environments where authentication info is set
// in <installPath>/conf/gopherbot.yaml, but everything else (plugins & config)
// is configured under <localPath>.

// Required indicates whether to return an error if neither file is found.
func getConfigFile(filename string, required bool, c interface{}) error {
	var (
		cf  []byte
		err error
	)

	loaded := false

	cf, err = ioutil.ReadFile(robot.installPath + "/conf/" + filename)
	if err == nil {
		if err = yaml.Unmarshal(cf, c); err != nil {
			err = fmt.Errorf("Unmarshalling installed \"%s\": %v", filename, err)
			Log(Error, err)
			return err
		}
		Log(Debug, fmt.Sprintf("Loaded installed conf/%s", filename))
		loaded = true
	}
	cf, err = ioutil.ReadFile(robot.localPath + "/conf/" + filename)
	if err != nil {
		err = fmt.Errorf("Reading local configuration for \"%s\": %v", filename, err)
		Log(Debug, err)
	} else {
		if err = yaml.Unmarshal(cf, c); err != nil {
			err = fmt.Errorf("Unmarshalling local \"%s\": %v", filename, err)
			Log(Error, err)
			return err // If a badly-formatted config is loaded, we always return an error
		}
		Log(Debug, fmt.Sprintf("Loaded configured conf/%s", filename))
		loaded = true
	}
	if required && !loaded {
		return err
	}
	return nil
}

// loadConfig loads the 'bot's json configuration files.
func loadConfig() error {
	var loglevel LogLevel
	newconfig := &botconf{}
	configload := make(map[string]json.RawMessage)
	pluginsOk := true

	if err := getConfigFile("gopherbot.yaml", true, &configload); err != nil {
		return fmt.Errorf("Loading configuration file: %v", err)
	}

	for key, value := range configload {
		var strval string
		var sarrval []string
		var epval []externalPlugin
		var mailval botMailer
		var boolval bool
		var intval int
		var val interface{}
		skip := false
		switch key {
		case "AdminContact", "Email", "Protocol", "Brain", "DefaultElevator", "DefaultAuthorizer", "Name", "Alias", "LogLevel":
			val = &strval
		case "DefaultAllowDirect":
			val = &boolval
		case "LocalPort":
			val = &intval
		case "ExternalPlugins":
			val = &epval
		case "DefaultChannels", "IgnoreUsers", "JoinChannels", "AdminUsers":
			val = &sarrval
		case "MailConfig":
			val = &mailval
		case "ProtocolConfig", "BrainConfig":
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
		case "DefaultElevator":
			newconfig.DefaultElevator = *(val.(*string))
		case "DefaultAuthorizer":
			newconfig.DefaultAuthorizer = *(val.(*string))
		case "Name":
			newconfig.Name = *(val.(*string))
		case "DefaultAllowDirect":
			newconfig.DefaultAllowDirect = *(val.(*bool))
		case "DefaultChannels":
			newconfig.DefaultChannels = *(val.(*[]string))
		case "IgnoreUsers":
			newconfig.IgnoreUsers = *(val.(*[]string))
		case "JoinChannels":
			newconfig.JoinChannels = *(val.(*[]string))
		case "ExternalPlugins":
			newconfig.ExternalPlugins = *(val.(*[]externalPlugin))
		case "AdminUsers":
			newconfig.AdminUsers = *(val.(*[]string))
		case "Alias":
			newconfig.Alias = *(val.(*string))
		case "LocalPort":
			newconfig.LocalPort = *(val.(*int))
		case "LogLevel":
			newconfig.LogLevel = *(val.(*string))
		}
	}

	loglevel = logStrToLevel(newconfig.LogLevel)
	setLogLevel(loglevel)

	robot.Lock()

	robot.defaultAllowDirect = newconfig.DefaultAllowDirect // defaults to false
	if newconfig.AdminContact != "" {
		robot.adminContact = newconfig.AdminContact
	}
	if newconfig.Email != "" {
		robot.email = newconfig.Email
	}
	robot.mailConf = newconfig.MailConfig
	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		if !strings.ContainsRune(string(aliases+escapeAliases), alias) {
			robot.Unlock()
			return fmt.Errorf("Invalid alias specified, ignoring. Must be one of: %s%s", escapeAliases, aliases)
		}
		robot.alias = alias
	}
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

	if newconfig.Protocol != "" {
		robot.protocol = newconfig.Protocol
	} else {
		robot.Unlock()
		return fmt.Errorf("Protocol not specified in gopherbot.yaml")
	}
	if newconfig.ProtocolConfig != nil {
		protocolConfig = newconfig.ProtocolConfig
	}

	if newconfig.AdminUsers != nil {
		robot.adminUsers = newconfig.AdminUsers
	}
	if newconfig.DefaultChannels != nil {
		robot.plugChannels = newconfig.DefaultChannels
	}
	if newconfig.ExternalPlugins != nil {
		for i, ep := range newconfig.ExternalPlugins {
			if len(ep.Name) == 0 || len(ep.Path) == 0 {
				pluginsOk = false
				Log(Error, fmt.Errorf("Reading external plugins, zero-length Name or Path for plugin #%d, not reloading plugins", i))
			}
		}
		if pluginsOk {
			robot.externalPlugins = newconfig.ExternalPlugins
		}
	}
	if newconfig.IgnoreUsers != nil {
		robot.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		robot.joinChannels = newconfig.JoinChannels
	}

	// loadPluginConfig does it's own locking
	robot.Unlock()

	globalLock.Lock()
	config = newconfig
	globalLock.Unlock()

	updateRegexes()
	if pluginsOk {
		loadPluginConfig()
	} else {
		return fmt.Errorf("Error reading external plugin config")
	}

	return nil
}
