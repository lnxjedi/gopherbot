package bot

import "fmt"

/* Handle incoming messages */

// interface Handler defines the callback API for Connectors
type Handler interface {
	ChannelMessage(channelName, userName, message string)
	DirectMessage(userName, message string)
	Log(l LogLevel, v ...interface{})
	// SetLogLevel updates the connector log level
	SetLogLevel(l LogLevel)
}

// ChannelMessage
func (b *Bot) ChannelMessage(channelName, userName, messageFull string) {
	// When command == true, the message was directed at the bot
	isCommand := false
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
	b.Log(Trace, fmt.Sprintf("Command \"%s\" in channel \"%s\"", message, channelName))
	b.handleMessage(isCommand, channelName, userName, message)
}

func (b *Bot) DirectMessage(userName, message string) {
	b.Log(Trace, "Direct message", message, "from user", userName)
	b.RLock()
	for _, user := range b.ignoreUsers {
		if userName == user {
			b.Log(Trace, "Ignoring user", userName)
			b.RUnlock()
			return
		}
	}
	b.RUnlock()
	b.handleMessage(true, "", userName, message)
}
