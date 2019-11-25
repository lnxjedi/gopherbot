// +build !module

package meme

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("goplugins/meme.so")
	bot.RegisterPlugin("meme", memehandler)
}
