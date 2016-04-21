package bot

import (
	"encoding/json"
	"fmt"
)

/* Handle incoming messages and other callbacks from the connector. */

// Handler is the interface that defines the callback API for Connectors
type Handler interface {
	// ChannelMessage is called by the connector for all messages the bot
	// can hear. The channelName and userName should be human-readable,
	// not internal representations.
	IncomingMessage(channelName, userName, message string)
	GetProtocolConfig() json.RawMessage
	SetName(n string)
	BotLogger
}

// ChannelMessage accepts an incoming channel message from the connector.
func (b *robot) IncomingMessage(channelName, userName, messageFull string) {
	// When command == true, the message was directed at the bot
	isCommand := false
	logChannel := channelName
	var message string

	b.RLock()
	for _, user := range b.ignoreUsers {
		if userName == user {
			b.Log(Debug, "Ignoring user", userName)
			b.RUnlock()
			return
		}
	}
	b.RUnlock()
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

// GetProtocolConfig returns the connector protocol's json.RawMessage to the connector
func (b *robot) GetProtocolConfig() json.RawMessage {
	var pc []byte
	b.RLock()
	// Make of copy of the protocol config for the plugin
	pc = append(pc, []byte(b.protocolConfig)...)
	b.RUnlock()
	return pc
}

// Connectors that support it can call SetName; otherwise it should
// be configured in gobot.conf.
func (b *robot) SetName(n string) {
	b.Lock()
	b.Log(Debug, "Setting name to: "+n)
	b.name = n
	b.Unlock()
	b.updateRegexes()
}
