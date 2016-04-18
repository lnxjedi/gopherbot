package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"unicode/utf8"
)

/* conf.go - methods and types for reading and storing json configuration */

// botconf specifies 'bot configuration, and is read from $GOBOT_CONFIGDIR/botconf.json
type botconf struct {
	Protocol        string          // Name of the connector protocol to use, e.g. "slack"
	Name            string          // Name of the 'bot, specify here if the protocol doesn't supply it (slack does)
	DefaultChannels []string        // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers     []string        // Users the 'bot never talks to - like other bots
	JoinChannels    []string        // Channels the 'bot should join when it logs in (not supported by all protocols)
	ExternalPlugins []string        // List of non-Go plugins to load from $GOBOT_CONFIGDIR/plugins/<pluginName>.json
	Alias           string          // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort       string          // Port number for listening on localhost, for CLI plugins
	LogLevel        string          // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
	ProtocolConfig  json.RawMessage // Protocol-specific configuration
}

// LoadConfig loads the 'bot's json configuration files. An error on first load
// results in log.fatal, but later Loads just log the error.
func (b *Bot) LoadConfig(configPath string) error {
	var (
		bc       []byte
		config   botconf
		loglevel LogLevel
	)

	bc, err := ioutil.ReadFile(configPath + "/gobot.json")
	if err != nil {
		return fmt.Errorf("Loading %s: %v", configPath+"/gobot.json", err)
	}
	if err := json.Unmarshal(bc, &config); err != nil {
		return fmt.Errorf("Unmarshalling JSON at %s: %v", configPath+"/gobot.json", err)
	}
	b.configPath = configPath

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
	b.SetLogLevel(loglevel)

	b.Lock()

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
	b.Unlock()
	if err := b.loadPluginConfig(); err != nil {
		return fmt.Errorf("Loading plugin configuration: %v", err)
	}

	return nil
}
