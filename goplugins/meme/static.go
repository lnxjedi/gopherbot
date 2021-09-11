package meme

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPreload("goplugins/meme.so")
	bot.RegisterPlugin("memes", memehandler)
}
