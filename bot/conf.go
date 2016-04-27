package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"unicode/utf8"
)

/* conf.go - methods and types for reading and storing json configuration */

// botconf specifies 'bot configuration, and is read from $GOPHER_LOCALDIR/botconf.json
type botconf struct {
	AdminContact    string          // Contact info for whomever administers the robot
	Protocol        string          // Name of the connector protocol to use, e.g. "slack"
	Name            string          // Name of the 'bot, specify here if the protocol doesn't supply it (slack does)
	DefaultChannels []string        // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers     []string        // Users the 'bot never talks to - like other bots
	JoinChannels    []string        // Channels the 'bot should join when it logs in (not supported by all protocols)
	ExternalPlugins []string        // List of non-Go plugins to load from $GOPHER_LOCALDIR/plugins/<pluginName>.json
	AdminUsers      []string        // List of users who can access administrative commands
	Alias           string          // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort       string          // Port number for listening on localhost, for CLI plugins
	LogLevel        string          // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
	ProtocolConfig  json.RawMessage // Protocol-specific configuration
}

// getConfigFile loads a configuration file first from installPath, then
// from localPath, allowing local config to override stock config
func (b *robot) getConfigFile(filename string, c interface{}) error {
	var (
		cf  []byte
		err error
		ok  bool
	)
	ok = false

	cf, err = ioutil.ReadFile(b.installPath + "/conf/" + filename)
	if err != nil {
		b.Log(Warn, fmt.Errorf("Loading stock configuration for \"%s\": %v", filename, err))
	} else {
		if err := json.Unmarshal(cf, c); err != nil {
			b.Log(Error, fmt.Errorf("Unmarshalling JSON for \"%s\": %v", filename, err))
		} else {
			ok = true
		}
	}

	cf, err = ioutil.ReadFile(b.localPath + "/conf/" + filename)
	if err != nil {
		b.Log(Debug, fmt.Errorf("Loading custom configuration for \"%s\": %v", filename, err))
	} else {
		if err := json.Unmarshal(cf, c); err != nil {
			b.Log(Error, fmt.Errorf("Unmarshalling JSON for plugin %s: %v", filename, err))
		} else {
			ok = true
		}
	}

	if !ok {
		return fmt.Errorf("No stock or local configuration found for %s", filename)
	}
	return nil
}

// loadConfig loads the 'bot's json configuration files. An error on first load
// results in log.fatal, but later Loads just log the error.
func (b *robot) loadConfig() error {
	var (
		config   botconf
		loglevel LogLevel
	)

	err := b.getConfigFile("gopherbot.json", &config)
	if err != nil {
		return err
	}

	switch config.LogLevel {
	case "trace":
		loglevel = Trace
	case "debug":
		loglevel = Debug
	case "info":
		loglevel = Info
	case "warn":
		loglevel = Warn
	default:
		loglevel = Error
	}
	b.setLogLevel(loglevel)

	b.lock.Lock()

	if config.AdminContact != "" {
		b.adminContact = config.AdminContact
	}
	if config.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(config.Alias)
		b.alias = alias
	}
	if config.LocalPort != "" {
		b.port = "127.0.0.1:" + config.LocalPort
		err := os.Setenv("GOPHER_HTTP_POST", "http://"+b.port)
		if err != nil {
			b.Log(Error, fmt.Errorf("Error exporting GOPHER_HTTP_PORT: ", err))
		}
	}
	if config.Name != "" {
		b.name = config.Name
	}

	if config.Protocol != "" {
		b.protocol = config.Protocol
	} else {
		return fmt.Errorf("Protocol not specified in gopherbot.json")
	}

	if config.AdminUsers != nil {
		b.adminUsers = config.AdminUsers
	}
	if config.DefaultChannels != nil {
		b.plugChannels = config.DefaultChannels
	}
	if config.ExternalPlugins != nil {
		b.externalPlugins = config.ExternalPlugins
	}
	if config.IgnoreUsers != nil {
		b.ignoreUsers = config.IgnoreUsers
	}
	if config.JoinChannels != nil {
		b.channels = config.JoinChannels
	}
	if config.ProtocolConfig != nil {
		b.protocolConfig = config.ProtocolConfig
	}

	// loadPluginConfig does it's own locking
	b.lock.Unlock()
	b.loadPluginConfig()

	return nil
}
