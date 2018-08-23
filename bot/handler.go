package bot

import (
	"encoding/json"
	"fmt"
	"strings"
)

// an empty object type for passing a Handler to the connector.
type handler struct{}

/* Handle incoming messages and other callbacks from the connector. */

// GetLogLevel returns the bot's current loglevel, mainly for the
// connector to make it's own decision about logging
func (h handler) GetLogLevel() LogLevel {
	return getLogLevel()
}

// GetLogToFile indicates to the terminal connector whether logging output is
// to a file, to prevent readline from redirecting log output.
func (h handler) GetLogToFile() bool {
	return logToFile
}

// GetInstallPath gets the path to the bot's install dir -
// the location of default configuration and stock external plugins.
func (h handler) GetInstallPath() string {
	return installPath
}

// GetConfigPath gets the path to the bot's (supposedly writable) configuration
// directory. This is the config path if specified, otherwise the install
// directory.
func (h handler) GetConfigPath() string {
	if len(configPath) > 0 {
		return configPath
	}
	return installPath
}

// ChannelMessage accepts an incoming channel message from the connector.
func (h handler) IncomingMessage(channelName, userName, messageFull string, raw interface{}) {
	Log(Trace, fmt.Sprintf("Incoming message '%s' in channel '%s'", messageFull, channelName))
	// When command == true, the message was directed at the bot
	isCommand := false
	logChannel := channelName
	var message string

	botCfg.RLock()
	for _, user := range botCfg.ignoreUsers {
		if strings.EqualFold(userName, user) {
			Log(Debug, "Ignoring user", userName)
			c := &botContext{User: userName}
			c.debug("robot is configured to ignore this user", true)
			emit(IgnoredUser)
			botCfg.RUnlock()
			return
		}
	}
	preRegex := botCfg.preRegex
	postRegex := botCfg.postRegex
	bareRegex := botCfg.bareRegex
	botCfg.RUnlock()
	if preRegex != nil {
		matches := preRegex.FindAllStringSubmatch(messageFull, -1)
		if matches != nil && len(matches[0]) == 2 {
			isCommand = true
			message = matches[0][1]
		}
	}
	if !isCommand && postRegex != nil {
		matches := postRegex.FindAllStringSubmatch(messageFull, -1)
		if matches != nil && len(matches[0]) == 3 {
			isCommand = true
			message = matches[0][1] + matches[0][2]
		}
	}
	if !isCommand {
		if bareRegex.MatchString(messageFull) {
			isCommand = true
		}
	}
	if !isCommand {
		message = messageFull
	}

	directMsg := false
	if len(channelName) == 0 { // true for direct messages
		isCommand = true
		directMsg = true
		logChannel = "(direct message)"
	}

	currentTasks.Lock()
	t := currentTasks.t
	nameMap := currentTasks.nameMap
	idMap := currentTasks.idMap
	nameSpaces := currentTasks.nameSpaces
	currentTasks.Unlock()
	confLock.RLock()
	repolist := repositories
	confLock.RUnlock()

	// Create the botContext and a goroutine to process the message and carry state,
	// which may eventually run a pipeline.
	c := &botContext{
		User:    userName,
		Channel: channelName,
		RawMsg:  raw,
		tasks: taskList{
			t:          t,
			nameMap:    nameMap,
			idMap:      idMap,
			nameSpaces: nameSpaces,
		},
		repositories:     repolist,
		isCommand:        isCommand,
		directMsg:        directMsg,
		msg:              message,
		workingDirectory: botCfg.workSpace,
		environment:      make(map[string]string),
	}
	Log(Debug, fmt.Sprintf("Message '%s' from user '%s' in channel '%s'; isCommand: %t", message, userName, logChannel, isCommand))
	c.debug(fmt.Sprintf("Message (command: %v) in channel %s: %s", isCommand, logChannel, message), true)
	go c.handleMessage()
}

// GetProtocolConfig unmarshals the connector's configuration data into a provided struct
func (h handler) GetProtocolConfig(v interface{}) error {
	botCfg.RLock()
	err := json.Unmarshal(protocolConfig, v)
	botCfg.RUnlock()
	return err
}

// GetBrainConfig unmarshals the brain's configuration data into a provided struct
func (h handler) GetBrainConfig(v interface{}) error {
	botCfg.RLock()
	err := json.Unmarshal(brainConfig, v)
	botCfg.RUnlock()
	return err
}

// GetHistoryConfig unmarshals the history provider's configuration data into a provided struct
func (h handler) GetHistoryConfig(v interface{}) error {
	botCfg.RLock()
	err := json.Unmarshal(historyConfig, v)
	botCfg.RUnlock()
	return err
}

// Log logs a message to the robot's log file (or stderr)
func (h handler) Log(l LogLevel, v ...interface{}) {
	Log(l, v...)
}

// Connectors that support it can call SetFullName; otherwise it can
// be configured in gopherbot.yaml.
func (h handler) SetFullName(n string) {
	Log(Debug, "Setting full name to: "+n)
	botCfg.Lock()
	botCfg.fullName = n
	botCfg.Unlock()
}

// Connectors that support it can call SetName; otherwise it should
// be configured in gopherbot.yaml.
func (h handler) SetName(n string) {
	Log(Debug, "Setting name to: "+n)
	botCfg.Lock()
	botCfg.name = n
	// Make sure the robot ignores messages from it's own name
	ignoring := false
	for _, name := range botCfg.ignoreUsers {
		if strings.EqualFold(n, name) {
			ignoring = true
			break
		}
	}
	if !ignoring {
		botCfg.ignoreUsers = append(botCfg.ignoreUsers, strings.ToLower(n))
	}
	botCfg.Unlock()
	updateRegexes()
}
