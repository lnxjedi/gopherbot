package bot

import (
	"fmt"
	"io/ioutil"
	"os"
	"unicode/utf8"

	"github.com/go-yaml/yaml"
)

/* conf.go - methods and types for reading and storing json configuration */

var protocolConfig, brainConfig []byte

// botconf specifies 'bot configuration, and is read from $GOPHER_LOCALDIR/botconf.json
type Botconf struct {
	AdminContact    string        // Contact info for whomever administers the robot
	Email           string        // From: address when the robot wants to send an email
	Protocol        string        // Name of the connector protocol to use, e.g. "slack"
	ProtocolConfig  yaml.MapSlice // Protocol-specific configuration, type for unmarshalling arbitrary config
	Brain           string        // Type of Brain to use
	BrainConfig     yaml.MapSlice // Brain-specific configuration, type for unmarshalling arbitrary config
	Name            string        // Name of the 'bot, specify here if the protocol doesn't supply it (slack does)
	DefaultChannels []string      // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers     []string      // Users the 'bot never talks to - like other bots
	JoinChannels    []string      // Channels the 'bot should join when it logs in (not supported by all protocols)
	ExternalPlugins []string      // List of non-Go plugins to load from $GOPHER_LOCALDIR/plugins/<pluginName>.json
	AdminUsers      []string      // List of users who can access administrative commands
	Alias           string        // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort       string        // Port number for listening on localhost, for CLI plugins
	LogLevel        string        // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
}

// getConfigFile loads a configuration file first from installPath, then
// from localPath, allowing local config to override stock config
func (b *robot) getConfigFile(filename string, c interface{}) error {
	var (
		cf  []byte
		err error
	)

	cf, err = ioutil.ReadFile(b.localPath + "/conf/" + filename)
	if err != nil {
		b.Log(Debug, fmt.Errorf("Reading custom configuration for \"%s\": %v", filename, err))
	} else {
		if err := yaml.Unmarshal(cf, c); err != nil {
			b.Log(Error, fmt.Errorf("Unmarshalling custom \"%s\": %v", filename, err))
		} else {
			return nil
		}
	}

	return fmt.Errorf("No local configuration found for %s", filename)
}

// loadConfig loads the 'bot's json configuration files. An error on first load
// results in log.fatal, but later Loads just log the error.
func (b *robot) loadConfig() error {
	var (
		config   Botconf
		loglevel LogLevel
	)

	// Load default config from const defaultConfig, then overlay
	// with yaml from <localdir>/conf/gopherbot.yaml
	if err := yaml.Unmarshal([]byte(defaultConfig), &config); err != nil {
		return fmt.Errorf("Unmarshalling robot default config: %v", err)
	}
	/*	if err := b.getConfigFile("gopherbot.yaml", &config); err != nil {
			return fmt.Errorf("Loading configuration file: %v", err)
	}*/
	full, _ := yaml.Marshal(config)
	fmt.Printf("Full:\n%s\n", string(full))
	os.Exit(0)

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
	if config.Email != "" {
		b.email = config.Email
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

	if config.Brain != "" {
		b.brainProvider = config.Brain
	}

	if config.Protocol != "" {
		b.protocol = config.Protocol
	} else {
		return fmt.Errorf("Protocol not specified in gopherbot.json")
	}

	// Re-marshal brainConfig and protocolConfig
	var err error
	if config.BrainConfig != nil {
		brainConfig, err = yaml.Marshal(config.BrainConfig)
		if err != nil {
			b.Log(Error, "Marshalling brainConfig: %v", err)
		}
	}
	if config.ProtocolConfig != nil {
		protocolConfig, err = yaml.Marshal(config.ProtocolConfig)
		if err != nil {
			b.Log(Error, "Marshalling protocolConfig: %v", err)
		}
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
		b.joinChannels = config.JoinChannels
	}

	// loadPluginConfig does it's own locking
	b.lock.Unlock()
	b.loadPluginConfig()

	return nil
}
