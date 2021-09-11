package duo

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPreload("goplugins/duo.so")
	bot.RegisterPlugin("duo", duohandler)
}
