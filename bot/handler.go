package bot

/* Handle incoming messages */

func (b *Bot) ChannelMsg(channelName, message string) {
	b.Debug("Message", message, "in channel", channelName)
}

func (b *Bot) DirectMsg(user, message string) {
	b.Debug("Direct message", message, "from user", user)
}
