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
	robot.RLock()
	defer robot.RUnlock()
	return robot.installPath
}

// GetConfigPath gets the path to the bot's (supposedly writable) configuration
// directory. This is the config path if specified, otherwise the install
// directory.
func (h handler) GetConfigPath() string {
	robot.RLock()
	defer robot.RUnlock()
	if len(robot.configPath) > 0 {
		return robot.configPath
	}
	return robot.installPath
}

// ChannelMessage accepts an incoming channel message from the connector.
func (h handler) IncomingMessage(channelName, userName, messageFull string, proto Protocol, raw interface{}) {
	Log(Trace, fmt.Sprintf("Incoming message '%s' in channel '%s'", messageFull, channelName))
	// When command == true, the message was directed at the bot
	isCommand := false
	logChannel := channelName
	var message string

	robot.RLock()
	for _, user := range robot.ignoreUsers {
		if strings.EqualFold(userName, user) {
			Log(Debug, "Ignoring user", userName)
			bot := &Robot{User: userName}
			bot.debug("", "robot is configured to ignore this user", true)
			emit(IgnoredUser)
			robot.RUnlock()
			return
		}
	}
	preRegex := robot.preRegex
	postRegex := robot.postRegex
	bareRegex := robot.bareRegex
	robot.RUnlock()
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

	// Create the Robot and a goroutine to process the message, which may
	// eventually run a plugin.
	bot := &Robot{
		User:      userName,
		Channel:   channelName,
		Protocol:  proto,
		RawMsg:    raw,
		isCommand: isCommand,
		directMsg: directMsg,
		msg:       message,
	}
	Log(Debug, fmt.Sprintf("Message '%s' from user '%s' in channel '%s'; isCommand: %t", message, userName, logChannel, isCommand))
	bot.debug("", fmt.Sprintf("Message (command: %v) in channel %s: %s", isCommand, logChannel, message), true)
	go bot.handleMessage()
}

// GetProtocolConfig unmarshals the connector's configuration data into a provided struct
func (h handler) GetProtocolConfig(v interface{}) error {
	robot.RLock()
	err := json.Unmarshal(protocolConfig, v)
	robot.RUnlock()
	return err
}

// GetBrainConfig unmarshals the brain's configuration data into a provided struct
func (h handler) GetBrainConfig(v interface{}) error {
	robot.RLock()
	err := json.Unmarshal(brainConfig, v)
	robot.RUnlock()
	return err
}

// GetElevateConfig unmarshals the brain's configuration data into a provided struct
func (h handler) GetElevateConfig(v interface{}) error {
	robot.RLock()
	err := json.Unmarshal(elevateConfig, v)
	robot.RUnlock()
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
	robot.Lock()
	robot.fullName = n
	robot.Unlock()
}

// Connectors that support it can call SetName; otherwise it should
// be configured in gobot.conf.
func (h handler) SetName(n string) {
	Log(Debug, "Setting name to: "+n)
	robot.Lock()
	robot.name = n
	// Make sure the robot ignores messages from it's own name
	ignoring := false
	for _, name := range robot.ignoreUsers {
		if strings.EqualFold(n, name) {
			ignoring = true
			break
		}
	}
	if !ignoring {
		robot.ignoreUsers = append(robot.ignoreUsers, strings.ToLower(n))
	}
	robot.Unlock()
	updateRegexes()
}
