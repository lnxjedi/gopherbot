package bot

import (
	"encoding/json"
	"fmt"
)

// handler struct hides robot methods that shouldn't be accessible
// to a connector.
type handler struct {
	bot *robot
}

/* Handle incoming messages and other callbacks from the connector. */

// Log exposes the private robot Log
func (h handler) Log(l LogLevel, v ...interface{}) {
	h.bot.Log(l, v)
}

// GetLogLevel returns the bot's current loglevel, mainly for the
// connector to make it's own decision about logging
func (h handler) GetLogLevel() LogLevel {
	h.bot.lock.RLock()
	l := h.bot.level
	h.bot.lock.RUnlock()
	return l
}

// ChannelMessage accepts an incoming channel message from the connector.
func (h handler) IncomingMessage(channelName, userName, messageFull string) {
	b := h.bot
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

// GetProtocolConfig unmarshals the connectors configuration data into a provided struct
func (h handler) GetProtocolConfig(v interface{}) error {
	b := h.bot
	b.lock.RLock()
	err := json.Unmarshal(b.protocolConfig, v)
	b.lock.RUnlock()
	return err
}

// Connectors that support it can call SetFullName; otherwise it can
// be configured in gobot.conf.
func (h handler) SetFullName(n string) {
	b := h.bot
	b.Log(Debug, "Setting full name to: "+n)
	b.lock.Lock()
	b.fullName = n
	b.lock.Unlock()
	b.updateRegexes()
}

// Connectors that support it can call SetName; otherwise it should
// be configured in gobot.conf.
func (h handler) SetName(n string) {
	b := h.bot
	b.Log(Debug, "Setting name to: "+n)
	b.lock.Lock()
	b.name = n
	b.lock.Unlock()
	b.updateRegexes()
}
