package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"unicode/utf8"
)

/* conf.go - methods and types for reading and storing json configuration */

// botconf specifies 'bot configuration, and is read from $GOPHER_LOCALDIR/botconf.json
type botconf struct {
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

// getConfigFile looks for configuration first in localPath/conf/, then installPath/conf/
func (b *robot) getConfigFile(name string) ([]byte, error) {
	var (
		cf  []byte
		err error
	)
	cf, err = ioutil.ReadFile(b.localPath + "/conf/" + name)
	if err != nil {
		cf, err = ioutil.ReadFile(b.installPath + "/conf/" + name)
		if err != nil {
			return nil, err
		}
		b.Log(Trace, fmt.Sprintf("Loaded stock configuration file %s", name))
	} else {
		b.Log(Trace, fmt.Sprintf("Loaded local configuration file %s", name))
	}
	return cf, nil
}

// loadConfig loads the 'bot's json configuration files. An error on first load
// results in log.fatal, but later Loads just log the error.
func (b *robot) loadConfig() error {
	var (
		bc       []byte
		config   botconf
		loglevel LogLevel
	)

	bc, err := b.getConfigFile("gopherbot.json")
	if err != nil {
		return fmt.Errorf("Loading %s: %v", b.localPath+"/gopherbot.json", err)
	}
	if err := json.Unmarshal(bc, &config); err != nil {
		return fmt.Errorf("Unmarshalling JSON at %s: %v", b.localPath+"/gopherbot.json", err)
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

	if len(config.Alias) > 0 {
		alias, _ := utf8.DecodeRuneInString(config.Alias)
		b.alias = alias
	}
	if config.LocalPort != "" {
		b.port = "127.0.0.1:" + config.LocalPort
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
	if err := b.loadPluginConfig(); err != nil {
		return fmt.Errorf("Loading plugin configuration: %v", err)
	}

	return nil
}
