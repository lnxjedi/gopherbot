package bot

import "fmt"

/* Handle incoming messages */

func (b *Bot) ChannelMsg(channelName, message string) {
	matched := false
	if b.preRegex != nil {
		matches := b.preRegex.FindAllStringSubmatch(message, 2)
		if matches != nil && len(matches[0]) == 3 {
			matched = true
			command := matches[0][2]
			b.SendChannelMessage(channelName, "I heard you! You said \""+command+"\"")
		}
	}
	if !matched && b.postRegex != nil {
		matches := b.postRegex.FindAllStringSubmatch(message, 2)
		b.Debug(fmt.Sprintf("%q", matches))
		if matches != nil && len(matches[0]) == 4 {
			matched = true
			command := matches[0][1] + matches[0][3]
			b.SendChannelMessage(channelName, "I heard you! You said \""+command+"\"")
		}
	}
	b.Debug("Message", message, "in channel", channelName)
}

func (b *Bot) DirectMsg(user, message string) {
	b.Debug("Direct message", message, "from user", user)
}
