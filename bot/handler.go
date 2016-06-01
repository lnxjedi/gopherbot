package bot

import (
	"encoding/json"
	"fmt"
)

/* Handle incoming messages and other callbacks from the connector. */

// GetLogLevel returns the bot's current loglevel, mainly for the
// connector to make it's own decision about logging
func (b *robot) GetLogLevel() LogLevel {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.level
}

// GetInstallPath gets the path to the bot's install dir -
// the location of default configuration and stock external plugins.
func (b *robot) GetInstallPath() string {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.installPath
}

// GetLocalPath gets the path to the bot's install dir -
// the location of default configuration and stock external plugins.
func (b *robot) GetLocalPath() string {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.localPath
}

// ChannelMessage accepts an incoming channel message from the connector.
func (b *robot) IncomingMessage(channelName, userName, messageFull string) {
	// When command == true, the message was directed at the bot
	isCommand := false
	logChannel := channelName
	var message string

	b.lock.RLock()
	for _, user := range b.ignoreUsers {
		if userName == user {
			b.Log(Debug, "Ignoring user", userName)
			b.lock.RUnlock()
			return
		}
	}
	b.lock.RUnlock()
	if b.preRegex != nil {
		matches := b.preRegex.FindAllStringSubmatch(messageFull, 2)
		if matches != nil && len(matches[0]) == 3 {
			isCommand = true
			message = matches[0][2]
		}
	}
	if !isCommand && b.postRegex != nil {
		matches := b.postRegex.FindAllStringSubmatch(messageFull, 2)
		if matches != nil && len(matches[0]) == 4 {
			isCommand = true
			message = matches[0][1] + matches[0][3]
		}
	}
	if !isCommand {
		message = messageFull
	}
	if len(channelName) == 0 { // true for direct messages
		isCommand = true
		logChannel = "(direct message)"
	}
	b.Log(Trace, fmt.Sprintf("Command \"%s\" in channel \"%s\"", message, logChannel))
	b.handleMessage(isCommand, channelName, userName, message)
}

// GetProtocolConfig unmarshals the connector's configuration data into a provided struct
func (b *robot) GetProtocolConfig(v interface{}) error {
	b.lock.RLock()
	err := json.Unmarshal(protocolConfig, v)
	b.lock.RUnlock()
	return err
}

// GetBrainConfig unmarshals the brain's configuration data into a provided struct
func (b *robot) GetBrainConfig(v interface{}) error {
	b.lock.RLock()
	err := json.Unmarshal(brainConfig, v)
	b.lock.RUnlock()
	return err
}

// Connectors that support it can call SetFullName; otherwise it can
// be configured in gobot.conf.
func (b *robot) SetFullName(n string) {
	b.Log(Debug, "Setting full name to: "+n)
	b.lock.Lock()
	b.fullName = n
	b.lock.Unlock()
	b.updateRegexes()
}

// Connectors that support it can call SetName; otherwise it should
// be configured in gobot.conf.
func (b *robot) SetName(n string) {
	b.Log(Debug, "Setting name to: "+n)
	b.lock.Lock()
	b.name = n
	b.lock.Unlock()
	b.updateRegexes()
}
