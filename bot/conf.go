package bot

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/lnxjedi/gopherbot/robot"
)

/* conf.go - methods and types for reading and storing json configuration */

var protocolConfig, brainConfig, historyConfig json.RawMessage

var protocolConfigs = struct {
	sync.RWMutex
	m map[string]json.RawMessage
}{
	m: map[string]json.RawMessage{},
}

func setProtocolConfigs(configs map[string]json.RawMessage) {
	protocolConfigs.Lock()
	defer protocolConfigs.Unlock()
	protocolConfigs.m = make(map[string]json.RawMessage, len(configs))
	for protocol, cfg := range configs {
		p := normalizeProtocolName(protocol)
		if p == "" || cfg == nil {
			continue
		}
		protocolConfigs.m[p] = cfg
	}
}

func getProtocolConfigFor(protocol string) json.RawMessage {
	p := normalizeProtocolName(protocol)
	protocolConfigs.RLock()
	cfg, ok := protocolConfigs.m[p]
	protocolConfigs.RUnlock()
	if ok {
		return applyProtocolRuntimeOverrides(p, cfg)
	}
	return applyProtocolRuntimeOverrides(p, protocolConfig)
}

func applyProtocolRuntimeOverrides(protocol string, cfg json.RawMessage) json.RawMessage {
	if cfg == nil {
		return nil
	}
	if protocol == "ssh" && sshPortOverride > 0 {
		var protocolCfg map[string]interface{}
		if err := json.Unmarshal(cfg, &protocolCfg); err != nil {
			Log(robot.Error, "Unable to apply SSH listen port override to protocol config: %v", err)
			return cfg
		}
		protocolCfg["ListenPort"] = sshPortOverride
		overridden, err := json.Marshal(protocolCfg)
		if err != nil {
			Log(robot.Error, "Unable to serialize protocol config with SSH listen port override: %v", err)
			return cfg
		}
		return overridden
	}
	return cfg
}

// ConfigLoader defines 'bot configuration, and is read from conf/robot.yaml
// Digested content ends up in currentCfg, see bot_process.go.
// ConfigLoader represents the structure of robot.yaml for validation.
type ConfigLoader struct {
	AdminContact         string                  `yaml:"AdminContact"`         // Contact info for whomever administers the robot
	MailConfig           botMailer               `yaml:"MailConfig"`           // Configuration for sending email
	PrimaryProtocol      string                  `yaml:"PrimaryProtocol"`      // Name of the primary connector protocol to use
	DefaultProtocol      string                  `yaml:"DefaultProtocol"`      // Protocol used when outbound message flow has no inbound protocol context
	SecondaryProtocols   []string                `yaml:"SecondaryProtocols"`   // Additional connector protocols to initialize when multi-protocol runtime is enabled
	BotInfo              *UserInfo               `yaml:"BotInfo"`              // Information about the robot
	UserRoster           []UserRosterEntry       `yaml:"UserRoster"`           // Global user directory entries; UserID accepted for legacy compatibility parsing
	ChannelRoster        []ChannelInfo           `yaml:"ChannelRoster"`        // List of channels mapping names to IDs
	Brain                string                  `yaml:"Brain"`                // Type of Brain to use
	EncryptionKey        string                  `yaml:"EncryptionKey"`        // Used to decrypt the "real" encryption key
	HistoryProvider      string                  `yaml:"HistoryProvider"`      // Name of provider to use for storing and retrieving job/plugin histories
	HttpDebug            bool                    `yaml:"HttpDebug"`            // Whether to turn on debug logging of local http API calls
	WorkSpace            string                  `yaml:"WorkSpace"`            // Read/Write area the robot uses to do work
	DefaultElevator      string                  `yaml:"DefaultElevator"`      // Elevator plugin for ElevatedCommands and ElevateImmediateCommands
	DefaultAuthorizer    string                  `yaml:"DefaultAuthorizer"`    // Authorizer plugin for AuthorizedCommands, or when AuthorizeAllCommands = true
	DefaultMessageFormat string                  `yaml:"DefaultMessageFormat"` // How the robot formats outgoing messages; default: BasicMarkdown
	DefaultAllowDirect   bool                    `yaml:"DefaultAllowDirect"`   // Whether plugins are available in a DM by default
	IgnoreUnlistedUsers  bool                    `yaml:"IgnoreUnlistedUsers"`  // Drop messages unless user is in global UserRoster
	SecureParameters     bool                    `yaml:"SecureParameters"`     // Don't publish parameters as environment variables
	DefaultChannels      []string                `yaml:"DefaultChannels"`      // Channels where plugins are active by default, e.g., ["general", "random"]
	IgnoreUsers          []string                `yaml:"IgnoreUsers"`          // Users the bot never talks to - like other bots
	JoinChannels         []string                `yaml:"JoinChannels"`         // Channels the bot should join on login (not supported by all protocols)
	DefaultJobChannel    string                  `yaml:"DefaultJobChannel"`    // Where job status is posted by default
	TimeZone             string                  `yaml:"TimeZone"`             // For evaluating the hour in a job schedule
	ExternalJobs         map[string]TaskSettings `yaml:"ExternalJobs"`         // List of available jobs; config in conf/jobs/<jobname>.yaml
	ExternalPlugins      map[string]TaskSettings `yaml:"ExternalPlugins"`      // List of non-Go plugins to load; config in conf/plugins/<plugname>.yaml
	ExternalTasks        map[string]TaskSettings `yaml:"ExternalTasks"`        // List executables for pipeline addition (not as starters)
	GoJobs               map[string]TaskSettings `yaml:"GoJobs"`               // Settings for Go jobs; config in conf/jobs/<jobname>.yaml
	GoPlugins            map[string]TaskSettings `yaml:"GoPlugins"`            // Settings for Go plugins; config in conf/plugins/<plugname>.yaml
	GoTasks              map[string]TaskSettings `yaml:"GoTasks"`              // Settings for Go tasks
	NameSpaces           map[string]TaskSettings `yaml:"NameSpaces"`           // Namespaces for shared parameters & memory sharing
	ParameterSets        map[string]TaskSettings `yaml:"ParameterSets"`        // Named sets of parameters, e.g., GITHUB_TOKEN used multiple places
	ScheduledJobs        []ScheduledTask         `yaml:"ScheduledJobs"`        // See tasks.go
	AdminUsers           []string                `yaml:"AdminUsers"`           // List of users with access to administrative commands
	Alias                string                  `yaml:"Alias"`                // One-character alias for commands directed at the bot, e.g., ';open the pod bay doors'
	LocalPort            int                     `yaml:"LocalPort"`            // Port number for localhost listening for CLI plugins
	LogLevel             string                  `yaml:"LogLevel"`             // Initial log level, modifiable by plugins. Options: "trace," "debug," "info," "warn," "error"
	LogDest              string                  `yaml:"LogDest"`              // one of stderr, stdout, <filename>
}

func normalizeSecondaryProtocols(primary string, secondary []string) []string {
	out := make([]string, 0, len(secondary))
	seen := make(map[string]bool)
	p := strings.ToLower(strings.TrimSpace(primary))
	for _, s := range secondary {
		name := strings.TrimSpace(s)
		if len(name) == 0 {
			continue
		}
		key := strings.ToLower(name)
		if key == p || key == "terminal" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, name)
	}
	return out
}

func secondaryIncludesPrimary(primary string, secondary []string) bool {
	p := strings.ToLower(strings.TrimSpace(primary))
	if p == "" {
		return false
	}
	for _, s := range secondary {
		if strings.ToLower(strings.TrimSpace(s)) == p {
			return true
		}
	}
	return false
}

func secondaryIncludesProtocol(protocol string, secondary []string) bool {
	p := strings.ToLower(strings.TrimSpace(protocol))
	if p == "" {
		return false
	}
	for _, s := range secondary {
		if strings.ToLower(strings.TrimSpace(s)) == p {
			return true
		}
	}
	return false
}

func isValidRosterUserName(name string) bool {
	if len(strings.TrimSpace(name)) == 0 {
		return false
	}
	return strings.ToLower(name) == name
}

type protocolFileConfig struct {
	config map[string]json.RawMessage
}

func normalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func providerConfigDirectoryForKey(key string) (string, bool) {
	switch key {
	case "BrainConfig":
		return "brains", true
	case "HistoryConfig":
		return "history", true
	default:
		return "", false
	}
}

func roleLabel(role string) string {
	if role == "" {
		return "Protocol"
	}
	return strings.ToUpper(role[:1]) + role[1:]
}

func loadProviderFileData(providerType, providerName string, required bool) (json.RawMessage, bool, error) {
	p := normalizeProviderName(providerName)
	if p == "" {
		return nil, false, fmt.Errorf("invalid %s provider name: %q", providerType, providerName)
	}
	dir := strings.ToLower(strings.TrimSpace(providerType))
	expectedKey := ""
	switch dir {
	case "brains":
		expectedKey = "BrainConfig"
	case "history":
		expectedKey = "HistoryConfig"
	default:
		return nil, false, fmt.Errorf("invalid provider type: %q", providerType)
	}
	configFile := filepath.Join(dir, p+".yaml")
	cfg := make(map[string]json.RawMessage)
	if err := getConfigFile(configFile, required, cfg); err != nil {
		if required {
			return nil, false, fmt.Errorf("loading %s provider '%s' config from conf/%s: %v", providerType, p, configFile, err)
		}
		Log(robot.Warn, "Loading %s provider '%s' config from conf/%s: %v", providerType, p, configFile, err)
		return nil, false, nil
	}
	if len(cfg) == 0 {
		if required {
			return nil, false, fmt.Errorf("%s provider '%s' configured but no conf/%s found", providerType, p, configFile)
		}
		Log(robot.Warn, "%s provider '%s' configured but no conf/%s found", providerType, p, configFile)
		return nil, false, nil
	}
	raw, ok := cfg[expectedKey]
	if !ok || raw == nil {
		return nil, false, fmt.Errorf("%s provider '%s' has no %s in conf/%s", providerType, p, expectedKey, filepath.ToSlash(configFile))
	}
	return raw, true, nil
}

func loadProtocolFileData(newconfig *ConfigLoader, protocol, role string, required bool) (protocolFileConfig, bool, error) {
	p := normalizeProtocolName(protocol)
	label := roleLabel(role)
	if p == "" {
		return protocolFileConfig{}, false, fmt.Errorf("invalid %s protocol name: %q", role, protocol)
	}
	configFile := filepath.Join("protocols", p+".yaml")
	protocolConfig := make(map[string]json.RawMessage)
	if err := getConfigFile(configFile, required, protocolConfig); err != nil {
		if required {
			return protocolFileConfig{}, false, fmt.Errorf("loading %s protocol config from conf/%s: %v", role, configFile, err)
		}
		Log(robot.Warn, "Loading %s protocol config from conf/%s: %v", role, configFile, err)
		return protocolFileConfig{}, false, nil
	}
	if len(protocolConfig) == 0 {
		if required {
			return protocolFileConfig{}, false, fmt.Errorf("%s protocol '%s' configured but no conf/%s found", label, p, configFile)
		}
		Log(robot.Warn, "%s protocol '%s' configured but no conf/%s found", label, p, configFile)
		return protocolFileConfig{}, false, nil
	}
	if _, ok := protocolConfig["UserMap"]; ok {
		return protocolFileConfig{}, false, fmt.Errorf("invalid configuration key in conf/%s for %s protocol '%s': UserMap", filepath.ToSlash(configFile), role, p)
	}
	var channels []ChannelInfo
	if raw, ok := protocolConfig["ChannelRoster"]; ok {
		if err := json.Unmarshal(raw, &channels); err != nil {
			Log(robot.Error, "Unmarshalling ChannelRoster from conf/%s: %v", configFile, err)
		} else {
			for i := range channels {
				channels[i].protocol = p
			}
			newconfig.ChannelRoster = append(newconfig.ChannelRoster, channels...)
		}
	}
	return protocolFileConfig{
		config: protocolConfig,
	}, true, nil
}

// DirectoryUser is the global user directory entry from robot.yaml UserRoster.
// This structure intentionally has no protocol internal ID field.
type DirectoryUser struct {
	UserName  string `yaml:"UserName"`  // Name used for authorization and identity decisions
	Email     string `yaml:"Email"`     // For Get*Attribute()
	Phone     string `yaml:"Phone"`     // For Get*Attribute()
	FullName  string `yaml:"FullName"`  // For Get*Attribute()
	FirstName string `yaml:"FirstName"` // For Get*Attribute()
	LastName  string `yaml:"LastName"`  // For Get*Attribute()
	BotUser   bool   `yaml:"BotUser"`   // These users aren't checked against MessageMatchers/ambient messages and never fall-through to "catchalls"
}

// UserRosterEntry is used only for configuration loading compatibility.
// UserID is accepted for legacy configs but ignored by the engine.
type UserRosterEntry struct {
	UserName  string `yaml:"UserName"`  // Name used for authorization and identity decisions
	UserID    string `yaml:"UserID"`    // Legacy parse-only field kept for config compatibility
	Email     string `yaml:"Email"`     // For Get*Attribute()
	Phone     string `yaml:"Phone"`     // For Get*Attribute()
	FullName  string `yaml:"FullName"`  // For Get*Attribute()
	FirstName string `yaml:"FirstName"` // For Get*Attribute()
	LastName  string `yaml:"LastName"`  // For Get*Attribute()
	BotUser   bool   `yaml:"BotUser"`   // These users aren't checked against MessageMatchers/ambient messages and never fall-through to "catchalls"
}

func (u UserRosterEntry) toDirectoryUser() *DirectoryUser {
	return &DirectoryUser{
		UserName:  u.UserName,
		Email:     u.Email,
		Phone:     u.Phone,
		FullName:  u.FullName,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		BotUser:   u.BotUser,
	}
}

// UserInfo is a runtime protocol-specific identity record used in per-protocol
// user maps and bot identity state.
type UserInfo struct {
	UserName     string `yaml:"UserName"`  // Name that refers to the user in bot config files
	UserID       string `yaml:"UserID"`    // Internal protocol ID (connector-local metadata/provenance)
	Email        string `yaml:"Email"`     // For Get*Attribute()
	Phone        string `yaml:"Phone"`     // For Get*Attribute()
	FullName     string `yaml:"FullName"`  // For Get*Attribute()
	FirstName    string `yaml:"FirstName"` // For Get*Attribute()
	LastName     string `yaml:"LastName"`  // For Get*Attribute()
	protoMention string `yaml:"-"`         // Robot only, @(mention) string
	protocol     string `yaml:"-"`         // protocol the user was loaded from (primary/secondary)
	BotUser      bool   `yaml:"BotUser"`   // These users aren't checked against MessageMatchers/ambient messages and never fall-through to "catchalls"
}

func applyDirectoryUserToUserInfo(dst *UserInfo, src *DirectoryUser) {
	if dst == nil || src == nil {
		return
	}
	dst.Email = src.Email
	dst.Phone = src.Phone
	dst.FullName = src.FullName
	dst.FirstName = src.FirstName
	dst.LastName = src.LastName
	dst.BotUser = src.BotUser
}

// ChannelInfo maps channel IDs to channel names when the connector doesn't
// provide a sensible name for use in configuration files.
type ChannelInfo struct {
	ChannelName, ChannelID string // human-readable and protocol-internal channel representations
	protocol               string // protocol the channel was loaded from (primary/secondary)
}

type userChanMaps struct {
	userIDProto    map[string]map[string]*UserInfo
	directoryUser  map[string]bool
	user           map[string]*DirectoryUser // Current map of username to global directory entry
	userProto      map[string]map[string]*UserInfo
	channelID      map[string]*ChannelInfo // Current map of channel ID to ChannelInfo struct
	channel        map[string]*ChannelInfo // Current map of channel name to ChannelInfo struct
	channelIDProto map[string]map[string]*ChannelInfo
	channelProto   map[string]map[string]*ChannelInfo
}

var currentUCMaps = struct {
	ucmap *userChanMaps // pointer to current struct
	sync.Mutex
}{
	nil,
	sync.Mutex{},
}

// Protects the bot config
var confLock sync.RWMutex
var config *ConfigLoader

// loadConfig loads the 'bot's yaml configuration files.
func loadConfig(preConnect bool) error {
	raiseThreadPriv("loading configuration")
	if preConnect {
		Log(robot.Info, "Loading initial pre-connection configuration")
	} else {
		Log(robot.Info, "Loading full post-connect configuration")
	}
	newconfig := &ConfigLoader{}
	newconfig.ExternalJobs = make(map[string]TaskSettings)
	newconfig.ExternalPlugins = make(map[string]TaskSettings)
	newconfig.ExternalTasks = make(map[string]TaskSettings)
	configload := make(map[string]json.RawMessage)
	processed := &configuration{}

	if err := getConfigFile(robotConfigFileName, true, configload); err != nil {
		return fmt.Errorf("loading configuration file: %v", err)
	}

	explicitDefaultAllowDirect := false

	for key, value := range configload {
		var strval string
		var sarrval []string
		var urval []UserRosterEntry
		var bival *UserInfo
		var crval []ChannelInfo
		var tval map[string]TaskSettings
		var stval []ScheduledTask
		var mailval botMailer
		var boolval bool
		var intval int
		var val interface{}
		skip := false
		switch key {
		case "AdminContact", "Email", "PrimaryProtocol", "DefaultProtocol", "Brain", "EncryptionKey", "HistoryProvider", "WorkSpace", "DefaultJobChannel", "DefaultElevator", "DefaultAuthorizer", "DefaultMessageFormat", "Name", "Alias", "LogDest", "LogLevel", "TimeZone":
			val = &strval
		case "DefaultAllowDirect", "HttpDebug", "IgnoreUnlistedUsers", "SecureParameters":
			val = &boolval
		case "BotInfo":
			val = &bival
		case "UserRoster":
			val = &urval
		case "ChannelRoster":
			val = &crval
		case "LocalPort":
			val = &intval
		case "ExternalJobs", "ExternalPlugins", "ExternalTasks", "GoJobs", "GoPlugins", "GoTasks", "NameSpaces", "ParameterSets":
			val = &tval
		case "ScheduledJobs":
			val = &stval
		case "DefaultChannels", "IgnoreUsers", "JoinChannels", "AdminUsers", "SecondaryProtocols":
			val = &sarrval
		case "MailConfig":
			val = &mailval
		case "BrainConfig", "HistoryConfig":
			targetDir, _ := providerConfigDirectoryForKey(key)
			err := fmt.Errorf("invalid configuration key in %s: %s (move to conf/%s/<provider>.yaml)", robotConfigFileName, key, targetDir)
			Log(robot.Error, err.Error())
			return err
		default:
			err := fmt.Errorf("invalid configuration key in %s: %s", robotConfigFileName, key)
			Log(robot.Error, err.Error())
			return err
		}
		if !skip {
			if err := json.Unmarshal(value, val); err != nil {
				err = fmt.Errorf("unmarshalling bot config value \"%s\": %v", key, err)
				Log(robot.Error, err.Error())
				return err
			}
		}
		switch key {
		case "AdminContact":
			newconfig.AdminContact = *(val.(*string))
		case "BotInfo":
			newconfig.BotInfo = *(val.(**UserInfo))
		case "MailConfig":
			newconfig.MailConfig = *(val.(*botMailer))
		case "PrimaryProtocol":
			newconfig.PrimaryProtocol = *(val.(*string))
		case "DefaultProtocol":
			newconfig.DefaultProtocol = *(val.(*string))
		case "Brain":
			newconfig.Brain = *(val.(*string))
		case "EncryptionKey":
			newconfig.EncryptionKey = *(val.(*string))
		case "HistoryProvider":
			newconfig.HistoryProvider = *(val.(*string))
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
		case "UserRoster":
			newconfig.UserRoster = *(val.(*[]UserRosterEntry))
		case "ChannelRoster":
			newconfig.ChannelRoster = *(val.(*[]ChannelInfo))
		case "DefaultAllowDirect":
			newconfig.DefaultAllowDirect = *(val.(*bool))
			explicitDefaultAllowDirect = true
		case "DefaultChannels":
			newconfig.DefaultChannels = *(val.(*[]string))
		case "IgnoreUsers":
			newconfig.IgnoreUsers = *(val.(*[]string))
		case "JoinChannels":
			newconfig.JoinChannels = *(val.(*[]string))
		case "SecondaryProtocols":
			newconfig.SecondaryProtocols = *(val.(*[]string))
		case "HttpDebug":
			newconfig.HttpDebug = *(val.(*bool))
		case "IgnoreUnlistedUsers":
			newconfig.IgnoreUnlistedUsers = *(val.(*bool))
		case "SecureParameters":
			newconfig.SecureParameters = *(val.(*bool))
		case "ExternalPlugins":
			newconfig.ExternalPlugins = *(val.(*map[string]TaskSettings))
		case "ExternalJobs":
			newconfig.ExternalJobs = *(val.(*map[string]TaskSettings))
		case "ExternalTasks":
			newconfig.ExternalTasks = *(val.(*map[string]TaskSettings))
		case "GoPlugins":
			newconfig.GoPlugins = *(val.(*map[string]TaskSettings))
		case "GoJobs":
			newconfig.GoJobs = *(val.(*map[string]TaskSettings))
		case "GoTasks":
			newconfig.GoTasks = *(val.(*map[string]TaskSettings))
		case "NameSpaces":
			newconfig.NameSpaces = *(val.(*map[string]TaskSettings))
		case "ParameterSets":
			newconfig.ParameterSets = *(val.(*map[string]TaskSettings))
		case "ScheduledJobs":
			newconfig.ScheduledJobs = *(val.(*[]ScheduledTask))
		case "AdminUsers":
			newconfig.AdminUsers = *(val.(*[]string))
		case "Alias":
			newconfig.Alias = *(val.(*string))
		case "LocalPort":
			newconfig.LocalPort = *(val.(*int))
		case "LogLevel":
			newconfig.LogLevel = *(val.(*string))
		case "LogDest":
			newconfig.LogDest = *(val.(*string))
		case "TimeZone":
			newconfig.TimeZone = *(val.(*string))
		}
	}

	processed.ignoreUnlistedUsers = newconfig.IgnoreUnlistedUsers
	processed.secureParamRetrieve = newconfig.SecureParameters
	processed.httpDebug = newconfig.HttpDebug
	if strings.TrimSpace(newconfig.PrimaryProtocol) == "" {
		return fmt.Errorf("PrimaryProtocol not specified in %s", robotConfigFileName)
	}
	processed.protocol = normalizeProtocolName(newconfig.PrimaryProtocol)
	if !preConnect {
		if runtimePrimary, ok := getRuntimePrimaryProtocol(); ok && runtimePrimary != "" && !strings.EqualFold(runtimePrimary, processed.protocol) {
			Log(robot.Error, "PrimaryProtocol change on reload ignored (configured: '%s', active: '%s')", processed.protocol, runtimePrimary)
			processed.protocol = runtimePrimary
		}
	}
	if secondaryIncludesPrimary(processed.protocol, newconfig.SecondaryProtocols) {
		Log(robot.Warn, "SecondaryProtocols includes primary protocol '%s'; ignoring duplicate entry", processed.protocol)
	}
	if secondaryIncludesProtocol("terminal", newconfig.SecondaryProtocols) {
		Log(robot.Warn, "SecondaryProtocols includes unsupported protocol 'terminal'; ignoring it")
	}
	processed.secondaryProtocols = normalizeSecondaryProtocols(processed.protocol, newconfig.SecondaryProtocols)
	processed.defaultProtocol = normalizeProtocolName(newconfig.DefaultProtocol)
	if processed.defaultProtocol == "" {
		processed.defaultProtocol = processed.protocol
	} else if processed.defaultProtocol != processed.protocol && !secondaryIncludesProtocol(processed.defaultProtocol, processed.secondaryProtocols) {
		Log(robot.Warn, "DefaultProtocol '%s' is not primary or in SecondaryProtocols; falling back to PrimaryProtocol '%s'", newconfig.DefaultProtocol, processed.protocol)
		processed.defaultProtocol = processed.protocol
	}

	for i := range newconfig.ChannelRoster {
		newconfig.ChannelRoster[i].protocol = processed.protocol
	}
	perProtocolConfigs := make(map[string]json.RawMessage, len(processed.secondaryProtocols)+1)
	// Load primary protocol config from conf/protocols/<primary>.yaml and then
	// re-append robot.yaml channels so robot.yaml remains the override layer.
	robotPrimaryChannels := append([]ChannelInfo(nil), newconfig.ChannelRoster...)
	newconfig.ChannelRoster = make([]ChannelInfo, 0, len(robotPrimaryChannels))
	primaryProtocolConfigFile, loaded, err := loadProtocolFileData(newconfig, processed.protocol, "primary", true)
	if err != nil {
		return err
	}
	primaryConfigPath := filepath.Join("protocols", processed.protocol+".yaml")
	if !loaded {
		return fmt.Errorf("primary protocol '%s' configured but conf/%s did not load", processed.protocol, filepath.ToSlash(primaryConfigPath))
	}
	newconfig.ChannelRoster = append(newconfig.ChannelRoster, robotPrimaryChannels...)
	rawPrimaryProtocolConfig, ok := primaryProtocolConfigFile.config["ProtocolConfig"]
	if !ok || rawPrimaryProtocolConfig == nil {
		return fmt.Errorf("primary protocol '%s' has no ProtocolConfig in conf/%s", processed.protocol, filepath.ToSlash(primaryConfigPath))
	}
	perProtocolConfigs[processed.protocol] = rawPrimaryProtocolConfig
	secondaryConfigByProtocol := make(map[string]protocolFileConfig, len(processed.secondaryProtocols))
	for _, secondary := range processed.secondaryProtocols {
		if cfg, ok, err := loadProtocolFileData(newconfig, secondary, "secondary", false); err != nil {
			return err
		} else if ok {
			secondaryConfigByProtocol[normalizeProtocolName(secondary)] = cfg
		}
	}
	for _, secondary := range processed.secondaryProtocols {
		secondaryConfig, ok := secondaryConfigByProtocol[normalizeProtocolName(secondary)]
		if !ok {
			continue
		}
		if raw, ok := secondaryConfig.config["ProtocolConfig"]; ok {
			perProtocolConfigs[normalizeProtocolName(secondary)] = raw
		} else {
			Log(robot.Warn, "Secondary protocol '%s' has no ProtocolConfig in conf/protocols/%s.yaml", secondary, normalizeProtocolName(secondary))
		}
	}
	setProtocolConfigs(perProtocolConfigs)
	if newconfig.Brain != "" {
		processed.brainProvider = newconfig.Brain
		if cfg, loaded, err := loadProviderFileData("brains", newconfig.Brain, true); err != nil {
			return err
		} else if loaded {
			brainConfig = cfg
		} else {
			brainConfig = nil
		}
	} else {
		brainConfig = nil
	}
	if newconfig.HistoryProvider == "" {
		newconfig.HistoryProvider = "mem"
	}
	processed.historyProvider = newconfig.HistoryProvider
	if cfg, loaded, err := loadProviderFileData("history", newconfig.HistoryProvider, true); err != nil {
		return err
	} else if loaded {
		historyConfig = cfg
	} else {
		historyConfig = nil
	}

	if newconfig.Alias != "" {
		alias, _ := utf8.DecodeRuneInString(newconfig.Alias)
		if !strings.ContainsRune(string(aliases+escapeAliases), alias) {
			return fmt.Errorf("invalid alias specified, ignoring. Must be one of: %s%s", escapeAliases, aliases)
		}
		processed.alias = alias
	}

	if len(newconfig.DefaultMessageFormat) == 0 {
		processed.defaultMessageFormat = robot.BasicMarkdown
	} else {
		processed.defaultMessageFormat = setFormat(newconfig.DefaultMessageFormat)
	}

	if explicitDefaultAllowDirect {
		processed.defaultAllowDirect = newconfig.DefaultAllowDirect
	} else {
		processed.defaultAllowDirect = true // rare case of defaulting to true
	}

	if newconfig.AdminContact != "" {
		processed.adminContact = newconfig.AdminContact
	}

	if newconfig.TimeZone != "" {
		tz, err := time.LoadLocation(newconfig.TimeZone)
		if err == nil {
			Log(robot.Info, "Set timezone: %s", tz)
			processed.timeZone = tz
		} else {
			Log(robot.Error, "Parsing time zone '%s', using local time; error: %v", newconfig.TimeZone, err)
			processed.timeZone = nil
		}
	}

	if newconfig.BotInfo != nil {
		// Note that connector-supplied values are copied
		// when processed becomes current.
		processed.botinfo = *newconfig.BotInfo
	}
	processed.mailConf = newconfig.MailConfig

	if newconfig.DefaultJobChannel != "" {
		processed.defaultJobChannel = newconfig.DefaultJobChannel
	}

	if newconfig.DefaultElevator != "" {
		processed.defaultElevator = newconfig.DefaultElevator
	}

	if newconfig.DefaultAuthorizer != "" {
		processed.defaultAuthorizer = newconfig.DefaultAuthorizer
	}

	// Defaults to robot.Error if not set
	processed.logLevel = logStrToLevel(newconfig.LogLevel)
	setLogLevel(processed.logLevel)

	if newconfig.AdminUsers != nil {
		processed.adminUsers = newconfig.AdminUsers
	} else {
		processed.adminUsers = []string{}
	}
	if newconfig.DefaultChannels != nil {
		processed.plugChannels = newconfig.DefaultChannels
	}
	if newconfig.ExternalPlugins != nil {
		et := make([]TaskSettings, 0)
		for name, task := range newconfig.ExternalPlugins {
			if task.Disabled {
				continue
			}
			task.Name = name
			// Plugins default to unprivileged
			if task.Privileged == nil {
				p := false
				task.Privileged = &p
			}
			et = append(et, task)
		}
		processed.externalPlugins = et
	}
	if newconfig.ExternalJobs != nil {
		et := make([]TaskSettings, 0)
		for name, task := range newconfig.ExternalJobs {
			if task.Disabled {
				continue
			}
			task.Name = name
			// Jobs default to privileged
			if task.Privileged == nil {
				p := true
				task.Privileged = &p
			}
			et = append(et, task)
		}
		processed.externalJobs = et
	}
	if newconfig.ExternalTasks != nil {
		et := make([]TaskSettings, 0)
		for name, task := range newconfig.ExternalTasks {
			if task.Disabled {
				continue
			}
			task.Name = name
			if task.Privileged == nil {
				p := false
				task.Privileged = &p
			}
			et = append(et, task)
		}
		processed.externalTasks = et
	}
	// NOTE on Go tasks - we can't just skip a disabled task, since they're
	// enabled by default. Disabled: true needs to pass through so it's disabled
	// in taskconf.go
	if newconfig.GoTasks != nil {
		gt := make([]TaskSettings, 0, len(newconfig.GoTasks))
		for name, task := range newconfig.GoTasks {
			task.Name = name
			gt = append(gt, task)
		}
		processed.goTasks = gt
	}
	if newconfig.GoPlugins != nil {
		gt := make([]TaskSettings, 0, len(newconfig.GoPlugins))
		for name, task := range newconfig.GoPlugins {
			task.Name = name
			// Plugins default to unprivileged
			if task.Privileged == nil {
				p := false
				task.Privileged = &p
			}
			gt = append(gt, task)
		}
		processed.goPlugins = gt
	}
	if newconfig.GoJobs != nil {
		gt := make([]TaskSettings, 0, len(newconfig.GoJobs))
		for name, task := range newconfig.GoJobs {
			task.Name = name
			// Jobs default to privileged
			if task.Privileged == nil {
				p := true
				task.Privileged = &p
			}
			gt = append(gt, task)
		}
		processed.goJobs = gt
	}
	if newconfig.NameSpaces != nil {
		ns := make([]TaskSettings, 0, len(newconfig.NameSpaces))
		for name, nameSpace := range newconfig.NameSpaces {
			nameSpace.Name = name
			ns = append(ns, nameSpace)
		}
		processed.nsList = ns
	}
	if newconfig.ParameterSets != nil {
		ps := make([]TaskSettings, 0, len(newconfig.ParameterSets))
		for name, parameterSet := range newconfig.ParameterSets {
			parameterSet.Name = name
			ps = append(ps, parameterSet)
		}
		processed.psList = ps
	}
	st := make([]ScheduledTask, 0, len(newconfig.ScheduledJobs))
	for _, s := range newconfig.ScheduledJobs {
		if len(s.Name) == 0 || len(s.Schedule) == 0 {
			Log(robot.Error, "Zero-length Name (%s) or Schedule (%s) in ScheduledTask, skipping", s.Name, s.Schedule)
		} else {
			st = append(st, s)
		}
	}
	processed.ScheduledJobs = st
	if newconfig.IgnoreUsers != nil {
		processed.ignoreUsers = newconfig.IgnoreUsers
	}
	if newconfig.JoinChannels != nil {
		processed.joinChannels = newconfig.JoinChannels
	}

	ucmaps := userChanMaps{
		userIDProto:    make(map[string]map[string]*UserInfo),
		directoryUser:  make(map[string]bool),
		user:           make(map[string]*DirectoryUser),
		userProto:      make(map[string]map[string]*UserInfo),
		channelID:      make(map[string]*ChannelInfo),
		channel:        make(map[string]*ChannelInfo),
		channelIDProto: make(map[string]map[string]*ChannelInfo),
		channelProto:   make(map[string]map[string]*ChannelInfo),
	}
	if len(newconfig.UserRoster) > 0 {
		for i, user := range newconfig.UserRoster {
			if len(strings.TrimSpace(user.UserName)) == 0 {
				Log(robot.Error, "Empty UserName in UserRoster entry, ignoring")
			} else if !isValidRosterUserName(user.UserName) {
				Log(robot.Error, "Username contains uppercase letters (%s), ignoring", user.UserName)
			} else {
				du := newconfig.UserRoster[i].toDirectoryUser()
				if _, ok := ucmaps.user[du.UserName]; !ok {
					ucmaps.user[du.UserName] = du
				}
				ucmaps.directoryUser[du.UserName] = true
			}
		}
	}
	if len(newconfig.ChannelRoster) > 0 {
		for i, ch := range newconfig.ChannelRoster {
			if len(ch.ChannelName) == 0 || len(ch.ChannelID) == 0 {
				Log(robot.Error, "One of ChannelName/ChannelID empty (%s/%s), ignoring", ch.ChannelName, ch.ChannelID)
			} else {
				c := &newconfig.ChannelRoster[i]
				p := normalizeProtocolName(c.protocol)
				if p == "" {
					p = processed.protocol
				}
				c.protocol = p
				protoNameMap, ok := ucmaps.channelProto[p]
				if !ok {
					protoNameMap = map[string]*ChannelInfo{}
					ucmaps.channelProto[p] = protoNameMap
				}
				protoNameMap[c.ChannelName] = c
				protoIDMap, ok := ucmaps.channelIDProto[p]
				if !ok {
					protoIDMap = map[string]*ChannelInfo{}
					ucmaps.channelIDProto[p] = protoIDMap
				}
				protoIDMap[c.ChannelID] = c
				if _, ok := ucmaps.channel[c.ChannelName]; !ok {
					ucmaps.channel[c.ChannelName] = c
				}
				if _, ok := ucmaps.channelID[c.ChannelID]; !ok {
					ucmaps.channelID[c.ChannelID] = c
				}
			}
		}
	}
	currentUCMaps.Lock()
	currentUCMaps.ucmap = &ucmaps
	currentUCMaps.Unlock()

	h := handler{}
	if len(newconfig.WorkSpace) > 0 {
		if err := h.GetDirectory(newconfig.WorkSpace); err == nil {
			processed.workSpace = newconfig.WorkSpace
			Log(robot.Debug, "Setting workspace directory to '%s'", processed.workSpace)
		} else {
			Log(robot.Error, "Getting WorkSpace directory '%s', using '%s': %v", newconfig.WorkSpace, configPath, err)
		}
	}

	// Items only read at start-up, before multi-threaded
	if preConnect {
		if cfg, ok := perProtocolConfigs[processed.protocol]; ok {
			protocolConfig = cfg
		}

		if newconfig.EncryptionKey != "" {
			processed.encryptionKey = newconfig.EncryptionKey
			newconfig.EncryptionKey = "XXXXXX" // too short to be valid anyway
		}
		if newconfig.LocalPort != 0 {
			processed.port = fmt.Sprintf("%d", newconfig.LocalPort)
		} else {
			processed.port = "0"
		}
		if len(newconfig.LogDest) > 0 {
			processed.logDest = newconfig.LogDest
		}
		if !cliOp { // CLI operations don't need a real history
			registration, ok := historyProviderRegistration(newconfig.HistoryProvider)
			if !ok {
				Log(robot.Error, "No provider registered for history type: \"%s\", falling back to 'mem'", processed.historyProvider)
				newconfig.HistoryProvider = "mem"
				processed.historyProvider = "mem"
				registration, ok = historyProviderRegistration("mem")
				if !ok {
					return fmt.Errorf("no provider registered for default history type: \"mem\"")
				}
			}
			hp := registration.Provider(handler{})
			if hp == nil {
				Log(robot.Error, "History provider '%s' initialization returned nil, falling back to 'mem'", newconfig.HistoryProvider)
				newconfig.HistoryProvider = "mem"
				processed.historyProvider = "mem"
				hp = mhprovider(handler{})
			}
			if hp == nil {
				return fmt.Errorf("unable to initialize history provider")
			}
			interfaces.history = hp
			if newconfig.HistoryProvider != "mem" {
				// Initialize the memory provider as a last-ditch fallback
				mhprovider(handler{})
			}
		}
	} else {
		// We should never dump the brain key
		newconfig.EncryptionKey = "XXXXXX"
		// initJobs need to run before post-connect loadTaskConfig
		initJobs()
	}

	newList, err := loadTaskConfig(processed, preConnect)
	if err != nil {
		return err
	}

	// Configuration successfully loaded, apply changes

	// Note we always take the locks on global values regardless
	// of preConnect.

	// Update structs supplied to "dump robot"
	// Note that dump commands are only allowed for the
	// terminal connector.
	confLock.Lock()
	config = newconfig
	confLock.Unlock()

	currentCfg.Lock()
	processed.botinfo.protoMention = currentCfg.botinfo.protoMention
	currentCfg.configuration = processed
	currentCfg.taskList = newList
	currentCfg.Unlock()

	if !preConnect {
		reconcileSecondaryConnectorRuntimes(processed.secondaryProtocols)
		updateRegexes()
		scheduleTasks()
		initializePlugins()
	}

	return nil
}
