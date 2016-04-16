package memes

import (
	"github.com/parsley42/gobot/bot"
)

func memegen(bot bot.ChatBot, channel, user, command string, args ...string) error {
	bot.SendUserMessage(user, "Hey bub, I heard that!")
	return nil
}

func init() {
	bot.RegisterPlugin("memes", memegen)
}
