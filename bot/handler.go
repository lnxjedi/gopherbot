package bot

import (
	"fmt"
	"strings"

	"github.com/uva-its/yaml"
)

// an empty object type for passing a Handler to the connector.
type handler struct{}

/* Handle incoming messages and other callbacks from the connector. */

// GetLogLevel returns the bot's current loglevel, mainly for the
// connector to make it's own decision about logging
func (h handler) GetLogLevel() LogLevel {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.level
}

// GetInstallPath gets the path to the bot's install dir -
// the location of default configuration and stock external plugins.
func (h handler) GetInstallPath() string {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.installPath
}

// GetLocalPath gets the path to the bot's install dir -
// the location of default configuration and stock external plugins.
func (h handler) GetLocalPath() string {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.localPath
}

// ChannelMessage accepts an incoming channel message from the connector.
func (h handler) IncomingMessage(channelName, userName, messageFull string) {
	// When command == true, the message was directed at the bot
	isCommand := false
	logChannel := channelName
	var message string

	b.lock.RLock()
	for _, user := range b.ignoreUsers {
		if strings.EqualFold(userName, user) {
			Log(Debug, "Ignoring user", userName)
			b.lock.RUnlock()
			return
		}
	}
	b.lock.RUnlock()
	if b.preRegex != nil {
		matches := b.preRegex.FindAllStringSubmatch(messageFull, -1)
		if matches != nil && len(matches[0]) == 2 {
			isCommand = true
			message = matches[0][1]
		}
	}
	if !isCommand && b.postRegex != nil {
		matches := b.postRegex.FindAllStringSubmatch(messageFull, -1)
		if matches != nil && len(matches[0]) == 3 {
			isCommand = true
			message = matches[0][1] + matches[0][2]
		}
	}
	if !isCommand {
		message = messageFull
	}
	if len(channelName) == 0 { // true for direct messages
		isCommand = true
		logChannel = "(direct message)"
	}
	Log(Trace, fmt.Sprintf("Command \"%s\" in channel \"%s\"", message, logChannel))
	handleMessage(isCommand, channelName, userName, message)
}

// GetProtocolConfig unmarshals the connector's configuration data into a provided struct
func (h handler) GetProtocolConfig(v interface{}) error {
	b.lock.RLock()
	err := yaml.Unmarshal(protocolConfig, v)
	b.lock.RUnlock()
	return err
}

// GetBrainConfig unmarshals the brain's configuration data into a provided struct
func (h handler) GetBrainConfig(v interface{}) error {
	b.lock.RLock()
	err := yaml.Unmarshal(brainConfig, v)
	b.lock.RUnlock()
	return err
}

// Log logs a message to the robot's log file (or stderr)
func (h handler) Log(l LogLevel, v ...interface{}) {
	Log(l, v...)
}

// Connectors that support it can call SetFullName; otherwise it can
// be configured in gobot.conf.
func (h handler) SetFullName(n string) {
	Log(Debug, "Setting full name to: "+n)
	b.lock.Lock()
	b.fullName = n
	b.lock.Unlock()
	updateRegexes()
}

// Connectors that support it can call SetName; otherwise it should
// be configured in gobot.conf.
func (h handler) SetName(n string) {
	Log(Debug, "Setting name to: "+n)
	b.lock.Lock()
	b.name = n
	ignoring := false
	for _, name := range b.ignoreUsers {
		if strings.EqualFold(n, name) {
			ignoring = true
			break
		}
	}
	if !ignoring {
		b.ignoreUsers = append(b.ignoreUsers, strings.ToLower(n))
	}
	b.lock.Unlock()
	updateRegexes()
}
