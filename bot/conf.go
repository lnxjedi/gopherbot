package bot

import (
	"fmt"
	"io/ioutil"
	"os"
	"unicode/utf8"

	"github.com/parsley42/yaml"
)

/* conf.go - methods and types for reading and storing json configuration */

var protocolConfig, brainConfig []byte

type externalPlugin struct {
	Name, Path string // List of names and paths for external plugins; relative paths are searched first in installdir, then localdir
}

// botconf specifies 'bot configuration, and is read from $GOPHER_LOCALDIR/botconf.json
type botconf struct {
	AdminContact    string           // Contact info for whomever administers the robot
	Email           string           // From: address when the robot wants to send an email
	Protocol        string           // Name of the connector protocol to use, e.g. "slack"
	ProtocolConfig  yaml.MapSlice    // Protocol-specific configuration, type for unmarshalling arbitrary config
	Brain           string           // Type of Brain to use
	BrainConfig     yaml.MapSlice    // Brain-specific configuration, type for unmarshalling arbitrary config
	Name            string           // Name of the 'bot, specify here if the protocol doesn't supply it (slack does)
	DefaultChannels []string         // Channels where plugins are active by default, e.g. [ "general", "random" ]
	IgnoreUsers     []string         // Users the 'bot never talks to - like other bots
	JoinChannels    []string         // Channels the 'bot should join when it logs in (not supported by all protocols)
	ExternalPlugins []externalPlugin // List of non-Go plugins to load
	AdminUsers      []string         // List of users who can access administrative commands
	Alias           string           // One-character alias for commands directed at the 'bot, e.g. ';open the pod bay doors'
	LocalPort       string           // Port number for listening on localhost, for CLI plugins
	LogLevel        string           // Initial log level, can be modified by plugins. One of "trace" "debug" "info" "warn" "error"
}

func init() {
	yaml.SetPreserveFieldCase(true)
}

var config botconf

// getConfigFile loads a config file from localPath. Required indicates whether
// to return an error if the file isn't found.
func (b *robot) getConfigFile(filename string, required bool, c interface{}) error {
	var (
		cf  []byte
		err error
	)

	cf, err = ioutil.ReadFile(b.localPath + "/conf/" + filename)
	if err != nil {
		err = fmt.Errorf("Reading custom configuration for \"%s\": %v", filename, err)
		b.Log(Debug, err)
		if !required {
			return nil
		} else {
			return err
		}
	} else {
		if err := yaml.Unmarshal(cf, c); err != nil {
			err = fmt.Errorf("Unmarshalling custom \"%s\": %v", filename, err)
			b.Log(Error, err)
			return err // If a badly-formatted config is loaded, we always return an error
		} else {
			return nil
		}
	}
}

// loadConfig loads the 'bot's json configuration files. An error on first load
// results in log.fatal, but later Loads just log the error.
func (b *robot) loadConfig() error {
	var loglevel LogLevel
	var newconfig botconf

	// Load default config from const defaultConfig, then overlay
	// with yaml from <localdir>/conf/gopherbot.yaml
	if err := yaml.Unmarshal([]byte(defaultConfig), &newconfig); err != nil {
		return fmt.Errorf("Unmarshalling robot default newconfig: %v", err)
	}
	if err := b.getConfigFile("gopherbot.yaml", true, &newconfig); err != nil {
		return fmt.Errorf("Loading newconfiguration file: %v", err)
	}

	switch newconfig.LogLevel {
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

	if newconfig.AdminContact != "" {
		b.adminContact = newconfig.AdminContact
	}
	if newconfig.Email != "" {
		b.email = newconfig.Email
	}
	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		b.alias = alias
	}
	if newconfig.LocalPort != "" {
		b.port = "127.0.0.1:" + newconfig.LocalPort
		err := os.Setenv("GOPHER_HTTP_POST", "http://"+b.port)
		if err != nil {
			b.Log(Error, fmt.Errorf("Error exporting GOPHER_HTTP_PORT: ", err))
		}
	}
	if newconfig.Name != "" {
		b.name = newconfig.Name
	}

	if newconfig.Brain != "" {
		b.brainProvider = newconfig.Brain
	}

	if newconfig.Protocol != "" {
		b.protocol = newconfig.Protocol
	} else {
		return fmt.Errorf("Protocol not specified in gopherbot.json")
	}

	// Re-marshal brainConfig and protocolConfig
	var err error
	if newconfig.BrainConfig != nil {
		brainConfig, err = yaml.Marshal(newconfig.BrainConfig)
		if err != nil {
			b.Log(Error, "Marshalling brainConfig: %v", err)
		}
	}
	if newconfig.ProtocolConfig != nil {
		protocolConfig, err = yaml.Marshal(newconfig.ProtocolConfig)
		if err != nil {
			b.Log(Error, "Marshalling protocolConfig: %v", err)
		}
	}

	if newconfig.AdminUsers != nil {
		b.adminUsers = newconfig.AdminUsers
	}
	if newconfig.DefaultChannels != nil {
		b.plugChannels = newconfig.DefaultChannels
	}
	if newconfig.ExternalPlugins != nil {
		b.externalPlugins = newconfig.ExternalPlugins
	}
	if newconfig.IgnoreUsers != nil {
		b.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		b.joinChannels = newconfig.JoinChannels
	}

	// loadPluginConfig does it's own locking
	b.lock.Unlock()

	botLock.Lock()
	config = newconfig
	botLock.Unlock()

	b.loadPluginConfig()

	return nil
}
