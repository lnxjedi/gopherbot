package bot

import "fmt"

/* Handle incoming messages */

// interface Handler defines the callback API for Connectors
type Handler interface {
	ChannelMsg(channelName, userName, message string)
	DirectMsg(userName, message string)
	Log(l LogLevel, v ...interface{})
	// SetLogLevel updates the connector log level
	SetLogLevel(l LogLevel)
}

func (b *Bot) ChannelMsg(channelName, userName, message string) {
	matched := false
	var command string

	if b.preRegex != nil {
		matches := b.preRegex.FindAllStringSubmatch(message, 2)
		if matches != nil && len(matches[0]) == 3 {
			matched = true
			command = matches[0][2]
		}
	}
	if !matched && b.postRegex != nil {
		matches := b.postRegex.FindAllStringSubmatch(message, 2)
		if matches != nil && len(matches[0]) == 4 {
			matched = true
			command = matches[0][1] + matches[0][3]
		}
	}
	b.Log(Trace, fmt.Sprintf("Command \"%s\" in channel \"%s\"", command, channelName))
	b.RLock()
	for _, user := range b.ignoreUsers {
		if userName == user {
			b.Log(Trace, "Ignoring user", userName)
			b.RUnlock()
			return
		}
	}
	b.RUnlock()
	b.dispatch(channelName, userName, command)
}

func (b *Bot) DirectMsg(userName, message string) {
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
	b.dispatch("", userName, message)
}
