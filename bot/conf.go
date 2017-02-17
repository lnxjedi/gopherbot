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

var protocolConfig, brainConfig json.RawMessage

type externalPlugin struct {
	Name, Path string // List of names and paths for external plugins; relative paths are searched first in installdir, then localdir
}

// botconf specifies 'bot configuration, and is read from $GOPHER_CONFIGDIR/conf/gopherbot.yaml
type botconf struct {
	AdminContact    string           // Contact info for whomever administers the robot
	Email           string           // From: address when the robot wants to send an email
	MailConfig      botMailer        // configuration for sending email
	Protocol        string           // Name of the connector protocol to use, e.g. "slack"
	ProtocolConfig  json.RawMessage  // Protocol-specific configuration, type for unmarshalling arbitrary config
	Brain           string           // Type of Brain to use
	BrainConfig     json.RawMessage  // Brain-specific configuration, type for unmarshalling arbitrary config
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

var config botconf

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

	cf, err = ioutil.ReadFile(b.installPath + "/conf/" + filename)
	if err == nil {
		if err = yaml.Unmarshal(cf, c); err != nil {
			err = fmt.Errorf("Unmarshalling installed \"%s\": %v", filename, err)
			Log(Error, err)
			return err
		} else {
			Log(Debug, fmt.Sprintf("Loaded installed conf/%s", filename))
			loaded = true
		}
	}
	cf, err = ioutil.ReadFile(b.localPath + "/conf/" + filename)
	if err != nil {
		err = fmt.Errorf("Reading local configuration for \"%s\": %v", filename, err)
		Log(Debug, err)
	} else {
		if err = yaml.Unmarshal(cf, c); err != nil {
			err = fmt.Errorf("Unmarshalling local \"%s\": %v", filename, err)
			Log(Error, err)
			return err // If a badly-formatted config is loaded, we always return an error
		} else {
			Log(Debug, fmt.Sprintf("Loaded configured conf/%s", filename))
			loaded = true
		}
	}
	if required && !loaded {
		return err
	} else {
		return nil
	}
}

// loadConfig loads the 'bot's json configuration files.
func loadConfig() error {
	var loglevel LogLevel
	var newconfig botconf
	pluginsOk := true

	// Load default config from const defaultConfig, then overlay
	// with yaml from <localdir>/conf/gopherbot.yaml
	if err := yaml.Unmarshal([]byte(defaultConfig), &newconfig); err != nil {
		return fmt.Errorf("Unmarshalling robot default newconfig: %v", err)
	}
	if err := getConfigFile("gopherbot.yaml", true, &newconfig); err != nil {
		return fmt.Errorf("Loading newconfiguration file: %v", err)
	}

	loglevel = logStrToLevel(newconfig.LogLevel)
	setLogLevel(loglevel)

	b.lock.Lock()

	if newconfig.AdminContact != "" {
		b.adminContact = newconfig.AdminContact
	}
	if newconfig.Email != "" {
		b.email = newconfig.Email
	}
	b.mailConf = newconfig.MailConfig
	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		if !strings.ContainsRune(string(aliases+escape_aliases), alias) {
			return fmt.Errorf("Invalid alias specified, ignoring. Must be one of: %s%s", escape_aliases, aliases)
		} else {
			b.alias = alias
		}
	}
	if newconfig.LocalPort != "" {
		b.port = "127.0.0.1:" + newconfig.LocalPort
		err := os.Setenv("GOPHER_HTTP_POST", "http://"+b.port)
		if err != nil {
			Log(Error, fmt.Errorf("Error exporting GOPHER_HTTP_PORT: %q", err))
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

	if newconfig.BrainConfig != nil {
		brainConfig = newconfig.BrainConfig
	}
	if newconfig.ProtocolConfig != nil {
		protocolConfig = newconfig.ProtocolConfig
	}

	if newconfig.AdminUsers != nil {
		b.adminUsers = newconfig.AdminUsers
	}
	if newconfig.DefaultChannels != nil {
		b.plugChannels = newconfig.DefaultChannels
	}
	if newconfig.ExternalPlugins != nil {
		for i, ep := range newconfig.ExternalPlugins {
			if len(ep.Name) == 0 || len(ep.Path) == 0 {
				pluginsOk = false
				Log(Error, fmt.Errorf("Reading external plugins, zero-length Name or Path for plugin #%d, not reloading plugins", i))
			}
		}
		if pluginsOk {
			b.externalPlugins = newconfig.ExternalPlugins
		}
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

	updateRegexes()
	if pluginsOk {
		loadPluginConfig()
	} else {
		return fmt.Errorf("Error reading external plugin config")
	}

	return nil
}
