package meme

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPlugin("memes", memehandler)
}
